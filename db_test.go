package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"os"
	"testing"
	"time"
)

func testDB(t *testing.T, db NovelDatabase) {
	t.Helper()
	ctx := context.Background()
	n := &Novel{
		Title:       "testy mc test face",
		Author:      fmt.Sprintf("t-%d", time.Now().Unix()),
		Description: "desc",
	}

	id, err := db.AddNovel(ctx, n)
	if err != nil {
		t.Error(err)
	}
	n.ID = id
	n.Description = "new desc"
	if err := db.UpdateNovel(ctx, n); err != nil {
		t.Error(err)
	}
	gotNovel, err := db.GetNovel(ctx, id)
	if got, want := gotNovel.Description, n.Description; got != want {
		t.Error(err)
	}
	if err := db.DeleteNovel(ctx, id); err != nil {
		t.Error(err)
	}
	if _, err := db.GetNovel(ctx, id); err == nil {
		t.Error("want non-nil err")
	}
}

func TestMemoryDB(t *testing.T) {
	testDB(t, newMemoryDB())
}

func TestFireStoreDB(t *testing.T) {
	projectID := os.Getenv("GOLANG_SAMPLES_FIRESTORE_PROJECT")
	if projectID == "" {
		t.Skipf("GOLANG_SAMPLES_FIRESTORE_PROJECT not set")
	}
	ctx := context.Background()
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("firestore.NewClient: %v", err)
	}
	defer client.Close()

	db, err := newFirestoreDB(client)
	if err != nil {
		t.Fatalf("newFirestoreDB: %v", err)
	}
	testDB(t, db)
}
