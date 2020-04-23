package main

import (
	"context"
	"github.com/IkezawaYuki/go-novel-shelf/internal/webtest"
	"log"
	"os"
	"testing"
)

var (
	wt *webtest.W
	n  *Novelshelf

	testDBs = map[string]Novelshelf{}
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	projectID := os.Getenv("GOLANG_SAMPLES_PROJECT_ID")

	if projectID == "" {
		log.Println("GOLANG_SAMPLES_PROJECT_ID is not set. Skipping")
		return
	}

	memoryDB := newMemoryDB()
	testDBs["memory"] = memoryDB
	if firestoreProjectID := os.Getenv("GOLANG_SAMPLES_FIRESTORE_PROJECT"); firestoreProjectID != "" {
		projectID = firestoreProjectID

	}
}
