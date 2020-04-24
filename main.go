package main

import (
	"cloud.google.com/go/errorreporting"
	"cloud.google.com/go/firestore"
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	uuid "github.com/satori/go.uuid"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"runtime/debug"
)

var (
	listTmpl   = parseTemplate("list.html")
	editTmpl   = parseTemplate("edit.html")
	detailTmpl = parseTemplate("detail.html")
)

func main() {
	godotenv.Load(".env")
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		log.Fatal("GOOGLE_CLOUD_PROJECT must be set")
	}
	ctx := context.Background()

	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(client)
	db, err := newFirestoreDB(client)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(db)
	n, err := NewNovelshelf(projectID, db)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(n)
	n.registerHandlers()

	log.Printf("Listening on localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func (n *Novelshelf) registerHandlers() {
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/novels", http.StatusFound))

	r.Methods("GET").Path("/novels").
		Handler(appHandler(n.listHandler))
	r.Methods("GET").Path("/novels/add").
		Handler(appHandler(n.addFormHandler))
	r.Methods("GET").Path("/novels/{id:[0-9a-zA-Z_\\-]+}").
		Handler(appHandler(n.detailHandler))
	r.Methods("GET").Path("/novels/{id:[0-9a-zA-Z_\\-]+}/edit").
		Handler(appHandler(n.editFormHandler))

	r.Methods("POST").Path("/novels").
		Handler(appHandler(n.createHandler))
	r.Methods("POST", "PUT").Path("/novels/{id:[0-9a-zA-Z_\\-]+}").
		Handler(appHandler(n.updateHandler))
	r.Methods("POST").Path("/novels/{id:[0-9a-zA-Z_\\-]+}:delete").
		Handler(appHandler(n.deleteHandler))

	r.Methods("GET").Path("/_ah/health").HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("ok"))
		})

	//r.Methods("GET").Path("/logs").Handler(appHandler(n.))

	http.Handle("/", handlers.CombinedLoggingHandler(n.logWriter, r))
}

func (n *Novelshelf) listHandler(w http.ResponseWriter, r *http.Request) *appError {
	novels, err := n.DB.ListNovels(context.Background())
	if err != nil {
		return n.appErrorf(r, err, "could not list novels: %v", err)
	}
	return listTmpl.Execute(n, w, r, novels)
}

func (n *Novelshelf) novelFromRequest(r *http.Request) (*Novel, error) {
	log.Println("novelFromRequest invoked")
	id := mux.Vars(r)["id"]
	fmt.Println(id)
	novel, err := n.DB.GetNovel(context.Background(), id)
	if err != nil {
		return nil, fmt.Errorf("could not find book: %v", err)
	}
	return novel, nil
}

func (n *Novelshelf) detailHandler(w http.ResponseWriter, r *http.Request) *appError {
	novel, err := n.novelFromRequest(r)
	if err != nil {
		return n.appErrorf(r, err, "%v", err)
	}
	return detailTmpl.Execute(n, w, r, novel)
}

func (n *Novelshelf) addFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	return editTmpl.Execute(n, w, r, nil)
}

func (n *Novelshelf) editFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	novel, err := n.novelFromRequest(r)
	if err != nil {
		return n.appErrorf(r, err, "%v", err)
	}
	return editTmpl.Execute(n, w, r, novel)
}

func (n *Novelshelf) novelFromForm(r *http.Request) (*Novel, error) {
	ctx := r.Context()
	imageURL, err := n.uploadFileFromForm(ctx, r)
	if err != nil {
		return nil, fmt.Errorf("could not upload file: %v", err)
	}
	if imageURL == "" {
		imageURL = r.FormValue("imageURL")
	}
	novel := &Novel{
		Title:         r.FormValue("title"),
		Author:        r.FormValue("author"),
		PublishedDate: r.FormValue("publishedDate"),
		ImageURL:      imageURL,
		Description:   r.FormValue("description"),
	}

	return novel, nil
}

func (n *Novelshelf) uploadFileFromForm(ctx context.Context, r *http.Request) (url string, err error) {
	f, fh, err := r.FormFile("image")
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if n.StorageBucket == nil {
		return "", fmt.Errorf("strage bucket is missing - check novelshelf.go")
	}
	if _, err := n.StorageBucket.Attrs(ctx); err != nil {
		if err == storage.ErrBucketNotExist {
			return "", fmt.Errorf("bucket %q does not exist: check novelshelf.go", n.StorageBucketName)
		}
		return "", fmt.Errorf("could not get bucket: %v", err)
	}

	name := uuid.Must(uuid.NewV4()).String() + path.Ext(fh.Filename)

	w := n.StorageBucket.Object(name).NewWriter(ctx)
	w.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}
	w.ContentType = fh.Header.Get("Content-Type")
	w.CacheControl = "public, max-age=86400"

	if _, err := io.Copy(w, f); err != nil {
		return "", err
	}
	if err := w.Close(); err != nil {
		return "", err
	}

	const publicURL = "https://storage.googleapis.com/%s/%s"
	return fmt.Sprintf(publicURL, n.StorageBucketName, name), nil
}

func (n *Novelshelf) createHandler(w http.ResponseWriter, r *http.Request) *appError {
	ctx := r.Context()
	novel, err := n.novelFromForm(r)
	if err != nil {
		return n.appErrorf(r, err, "could not parse book from form: %v", err)
	}
	id, err := n.DB.AddNovel(ctx, novel)
	if err != nil {
		return n.appErrorf(r, err, "could not save novel: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/novels/%s", id), http.StatusFound)
	return nil
}

func (n *Novelshelf) updateHandler(w http.ResponseWriter, r *http.Request) *appError {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	if id != "" {
		return n.appErrorf(r, errors.New("no novel with empty ID"), "no novel with empty ID")
	}
	if id == "" {
		return n.appErrorf(r, errors.New("no book with empty ID"), "no book with empty ID")
	}

	novel, err := n.novelFromForm(r)
	if err != nil {
		return n.appErrorf(r, err, "could not parse novel from form: %v", err)
	}
	novel.ID = id

	err = n.DB.UpdateNovel(ctx, novel)
	if err != nil {
		return n.appErrorf(r, err, "could not save novel: %v", err)
	}
	http.Redirect(w, r, fmt.Sprintf("/novels/%s", novel.ID), http.StatusFound)
	return nil
}

func (n *Novelshelf) deleteHandler(w http.ResponseWriter, r *http.Request) *appError {
	ctx := r.Context()
	id := mux.Vars(r)["id"]
	err := n.DB.DeleteNovel(ctx, id)
	if err != nil {
		return n.appErrorf(r, err, "could not delete novel: %v", err)
	}
	http.Redirect(w, r, "/novels", http.StatusFound)
	return nil
}

type appHandler func(http.ResponseWriter, *http.Request) *appError

type appError struct {
	Error   error
	Message string
	Code    int
	Req     *http.Request
	Novel   *Novelshelf
	Stack   []byte
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		fmt.Fprintf(e.Novel.logWriter, "Handler error (reported to Error Reporting): status code: %d, message: %s,"+
			"underlying err: %v+\n", e.Code, e.Message, e.Error)
		w.WriteHeader(e.Code)
		e.Novel.errorClient.Report(errorreporting.Entry{
			Error: e.Error,
			Req:   e.Req,
			Stack: e.Stack,
		})
		e.Novel.errorClient.Flush()
	}
}

func (n *Novelshelf) appErrorf(r *http.Request, err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
		Req:     r,
		Novel:   n,
		Stack:   debug.Stack(),
	}
}
