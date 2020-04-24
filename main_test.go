package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"github.com/IkezawaYuki/go-novel-shelf/internal/webtest"
	"github.com/joho/godotenv"
	"io/ioutil"
	"log"
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

func TestNoNovel(t *testing.T) {
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
