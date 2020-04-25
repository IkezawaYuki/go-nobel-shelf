package main

import (
	"bytes"
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/IkezawaYuki/go-novel-shelf/internal/webtest"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

var (
	wt *webtest.W
	n  *Novelshelf

	testDBs = map[string]NovelDatabase{}
)

func TestMain(m *testing.M) {
	godotenv.Load(".env")

	ctx := context.Background()
	projectID := os.Getenv("GOLANG_SAMPLES_PROJECT_ID")

	if projectID == "" {
		log.Println("GOLANG_SAMPLES_PROJECT_ID is not set. Skipping")
		return
	}

	memoryDB := newMemoryDB()
	fmt.Println(memoryDB)
	testDBs["memory"] = memoryDB
	if firestoreProjectID := os.Getenv("GOLANG_SAMPLES_FIRESTORE_PROJECT"); firestoreProjectID != "" {
		projectID = firestoreProjectID
		client, err := firestore.NewClient(ctx, projectID)
		if err != nil {
			log.Fatalf("firestore.NewClient: %v", err)
		}

		docs, err := client.Collection("novels").DocumentRefs(ctx).GetAll()
		if err != nil {
			for _, d := range docs {
				if _, err := d.Delete(ctx); err != nil {
					log.Fatalf("delete: %v", err)
				}
			}
		}
		db, err := newFirestoreDB(client)
		if err != nil {
			log.Fatalf("newFirestroeDB: %v", err)
		}
		testDBs["firestore"] = db
	} else {
		log.Println("GOLANG_SAMPLES_FIRESTORE_PROJECT not set. Slipping Firestore database tests")
	}

	var err error
	n, err = NewNovelshelf(projectID, memoryDB)
	if err != nil {
		log.Fatalf("NewNovelshelf: %v", err)
	}
	log.SetOutput(ioutil.Discard)
	n.logWriter = ioutil.Discard

	serv := httptest.NewServer(nil)
	wt = webtest.New(nil, serv.Listener.Addr().String())

	n.registerHandlers()
	os.Exit(m.Run())
}

func TestNoNovels(t *testing.T) {
	for name, db := range testDBs {
		t.Run(name, func(t *testing.T) {
			n.DB = db
			bodyContains(t, wt, "/", "No novels found")
		})
	}
}

func TestNovelDetail(t *testing.T) {
	for name, db := range testDBs {
		t.Run(name, func(t *testing.T) {
			n.DB = db
			ctx := context.Background()
			const title = "novel mc novel"
			id, err := n.DB.AddNovel(ctx, &Novel{
				Title: title,
			})
			if err != nil {
				t.Fatal(err)
			}
			bodyContains(t, wt, "/", title)
			novelPath := fmt.Sprintf("/novels/%s", id)
			bodyContains(t, wt, novelPath, title)
			if err := n.DB.DeleteNovel(ctx, id); err != nil {
				t.Fatal(err)
			}
			bodyContains(t, wt, "/", "No novels found")
		})
	}
}

func TestEditNovel(t *testing.T) {
	for name, db := range testDBs {
		t.Run(name, func(t *testing.T) {
			n.DB = db
			ctx := context.Background()
			const title = "novel mc novel"
			id, err := n.DB.AddNovel(ctx, &Novel{
				Title: title,
			})
			if err != nil {
				t.Fatal(err)
			}
			novelPath := fmt.Sprintf("/novels/%s", id)
			editPath := novelPath + "/edit"
			bodyContains(t, wt, editPath, "Edit novel")
			bodyContains(t, wt, editPath, title)

			var body bytes.Buffer
			m := multipart.NewWriter(&body)
			m.WriteField("title", "simpsons")
			m.WriteField("author", "homer")
			m.Close()

			resp, err := wt.Post(novelPath, "multipart/form-data; boundary="+m.Boundary(), &body)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := resp.Request.URL.Path, novelPath; got != want {
				t.Errorf("got %s, want %s", got, want)
			}
			bodyContains(t, wt, novelPath, "simpsons")
			bodyContains(t, wt, novelPath, "homer")

			if err := n.DB.DeleteNovel(ctx, id); err != nil {
				t.Fatalf("got err %v, want nil", err)
			}
		})
	}
}

func TestAddAndDelete(t *testing.T) {
	for name, db := range testDBs {
		t.Run(name, func(t *testing.T) {
			n.DB = db
			bodyContains(t, wt, "/novels/add", "Add novel")
			novelPath := fmt.Sprintf("/novels")

			var body bytes.Buffer
			m := multipart.NewWriter(&body)
			m.WriteField("title", "simpsons")
			m.WriteField("author", "homer")

			m.Close()
			resp, err := wt.Post(novelPath, "multipart/form-data; boundary="+m.Boundary(), &body)
			if err != nil {
				t.Fatal(err)
			}
			gotPath := resp.Request.URL.Path
			if wantPrefix := "/novels"; !strings.HasPrefix(gotPath, wantPrefix) {
				t.Fatalf("redirect: got %q, want prefix %q", gotPath, wantPrefix)
			}
			bodyContains(t, wt, gotPath, "simpsons")
			bodyContains(t, wt, gotPath, "homer")

			_, err = wt.Post(gotPath+":delete", "", nil)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSendLog(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := n.logWriter
	n.logWriter = buf

	bodyContains(t, wt, "/logs", "Log sent!")

	n.logWriter = oldLogger
	if got, want := buf.String(), "Good job!"; !strings.Contains(got, want) {
		t.Errorf("/logs logged\n----\n%v\n----\nWant to contain:\n---\n%v", got, want)
	}
}

func TestSendError(t *testing.T) {
	buf := &bytes.Buffer{}
	oldLogger := n.logWriter
	n.logWriter = buf

	bodyContains(t, wt, "/errors", "Error Reporting")

	n.logWriter = oldLogger

	if got, want := buf.String(), "uh oh"; !strings.Contains(got, want) {
		t.Errorf("/errors logged\n----\n%v\n----\nWant to contain:\n----\n%v", got, want)
	}
}

func bodyContains(t *testing.T, wt *webtest.W, path, contains string) bool {
	t.Helper()
	body, _, err := wt.GetBody(path)
	if err != nil {
		t.Error(err)
		return false
	}
	if !strings.Contains(body, contains) {
		t.Errorf("got:\n----\n%s\nWant to contains:\n%s----", body, contains)
	}
	return true
}
