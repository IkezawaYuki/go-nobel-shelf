package infrastructure

import (
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"context"
	"encoding/json"
	"fmt"
	"github.com/IkezawaYuki/go-novel-shelf/domain"
	"github.com/gorilla/handlers"
	uuid "github.com/satori/go.uuid"
	"io"
	"os"
	"time"

	"github.com/gorilla/mux"
	"log"
	"net/http"
	"path"
	"strconv"
)

var (
	listTmpl   = parseTemplate("list.html")
	editTmpl   = parseTemplate("edit.html")
	detailTmpl = parseTemplate("detail.html")
)

type appHandler func(w http.ResponseWriter, r *http.Request) *appError

func Run() {
	RegisterHandlers()
	//appengine.Main()
}

func RegisterHandlers() {
	r := mux.NewRouter()

	r.Handle("/", http.RedirectHandler("/novels", http.StatusFound))

	r.Methods("GET").Path("/novels").Handler(appHandler(listHandler))
	r.Methods("GET").Path("/novels/mine").Handler(appHandler(listMineHandler))
	r.Methods("GET").Path("/novels/{id:[0-9]+}").Handler(appHandler(detailHandler))
	r.Methods("GET").Path("/novels/add").Handler(appHandler(addFormHandler))
	r.Methods("GET").Path("/novels/{id:[0-9]+}/edit").Handler(appHandler(editFormHandler))

	r.Methods("POST").Path("/novels").Handler(appHandler(createHandler))
	r.Methods("POST").Path("/novels/{id:[0-9]+}").Handler(appHandler(updateHandler))
	r.Methods("POST").Path("/novels/{id:[0-9]+}:delete").Handler(appHandler(deleteHandler))

	r.Methods("GET").Path("/_ah/health").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.Handle("/", handlers.CombinedLoggingHandler(os.Stderr, r))

	srv := &http.Server{
		Handler: r,
		Addr:    "127.0.0.1:8080",
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	srv.ListenAndServe()
}

func listHandler(w http.ResponseWriter, r *http.Request) *appError {
	fmt.Println("list handler is invoked")
	novels, err := DB.ListNovels()
	if err != nil {
		return appErrorf(err, "could not list novels: %v", err)
	}
	return listTmpl.Execute(w, r, novels)
}

func listMineHandler(w http.ResponseWriter, r *http.Request) *appError {
	user := profileFromSession(r)
	if user == nil {
		http.Redirect(w, r, "/login?redirect=/novel/mine", http.StatusFound)
		return nil
	}

	novels, err := DB.ListNovelsCreatedBy(user.ID)
	if err != nil {
		return appErrorf(err, "could not list novels: %v", err)
	}
	return listTmpl.Execute(w, r, novels)
}

func novelFromRequest(r *http.Request) (*domain.Novel, error) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("bad novel id: %v", err)
	}
	novel, err := DB.GetNovel(id)
	if err != nil {
		return nil, fmt.Errorf("could not find book: %v", err)
	}
	return novel, nil
}

func detailHandler(w http.ResponseWriter, r *http.Request) *appError {
	novel, err := novelFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return detailTmpl.Execute(w, r, novel)
}

func addFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	return editTmpl.Execute(w, r, nil)
}

func editFormHandler(w http.ResponseWriter, r *http.Request) *appError {
	novel, err := novelFromRequest(r)
	if err != nil {
		return appErrorf(err, "%v", err)
	}
	return editTmpl.Execute(w, r, novel)
}

func novelFromForm(r *http.Request) (*domain.Novel, error) {
	imageURL, err := uploadFileFromForm(r)
	if err != nil {
		return nil, fmt.Errorf("could not upload file: %v", err)
	}
	if imageURL == "" {
		imageURL = r.FormValue("imageURL")
	}
	novel := &domain.Novel{
		Title:         r.FormValue("title"),
		Author:        r.FormValue("author"),
		PublishedDate: r.FormValue("publishedDate"),
		ImageURL:      imageURL,
		Description:   r.FormValue("description"),
		CreatedBy:     r.FormValue("createdBy"),
		CreatedByID:   r.FormValue("createdByID"),
	}

	if novel.CreatedByID == "" {
		user := profileFromSession(r)
		if user != nil {
			novel.CreatedBy = user.DisplayName
			novel.CreatedByID = user.ID
		} else {
			novel.SetCreatorAnonymous()
		}
	}
	return novel, nil
}

func uploadFileFromForm(r *http.Request) (url string, err error) {
	f, fh, err := r.FormFile("image")
	if err == http.ErrMissingFile {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if StorageBucket == nil {
		return "", fmt.Errorf("strage bucket is missing - check config.go")
	}
	name := uuid.Must(uuid.NewV4()).String() + path.Ext(fh.Filename)
	ctx := context.Background()
	w := StorageBucket.Object(name).NewWriter(ctx)
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
	return fmt.Sprintf(publicURL, StorageBucketName, name), nil
}

func createHandler(w http.ResponseWriter, r *http.Request) *appError {
	novel, err := novelFromForm(r)
	if err != nil {
		return appErrorf(err, "could not parse book from form: %v", err)
	}
	id, err := DB.AddNovel(novel)
	if err != nil {
		return appErrorf(err, "could not save novel: %v", err)
	}
	go publishedUpdate(id)
	http.Redirect(w, r, fmt.Sprintf("/novels/%d", id), http.StatusFound)
	return nil
}

func updateHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "bad novel id: %v", err)
	}
	novel, err := novelFromForm(r)
	if err != nil {
		return appErrorf(err, "could not parse novel from form: %v", err)
	}
	novel.ID = id

	err = DB.UpdateBook(novel)
	if err != nil {
		return appErrorf(err, "could not save novel: %v", err)
	}
	go publishedUpdate(novel.ID)
	http.Redirect(w, r, fmt.Sprintf("/novels/%d", novel.ID), http.StatusFound)
	return nil
}

func deleteHandler(w http.ResponseWriter, r *http.Request) *appError {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		return appErrorf(err, "bad novel id: %v", err)
	}
	err = DB.DeleteNovel(id)
	if err != nil {
		return appErrorf(err, "could not delete novel: %v", err)
	}
	http.Redirect(w, r, "/novels", http.StatusFound)
	return nil
}

func publishedUpdate(novelID int64) {
	if PubsubClient == nil {
		return
	}
	ctx := context.Background()
	b, err := json.Marshal(novelID)
	if err != nil {
		return
	}
	topic := PubsubClient.Topic(PubsubTopicID)
	_, err = topic.Publish(ctx, &pubsub.Message{
		Data: b,
	}).Get(ctx)
	log.Printf("Published update to Pub/Sub for Novel for Novel ID %d: %v", novelID, err)
}

type appError struct {
	Error   error
	Message string
	Code    int
}

func (fn appHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil {
		log.Printf("Handler error: status code: %d, message: %s, underlying err: %#v",
			e.Code, e.Message, e.Error)
	}
}

func appErrorf(err error, format string, v ...interface{}) *appError {
	return &appError{
		Error:   err,
		Message: fmt.Sprintf(format, v...),
		Code:    500,
	}
}
