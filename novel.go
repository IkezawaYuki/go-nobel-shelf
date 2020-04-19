package main

import (
	"cloud.google.com/go/errorreporting"
	"cloud.google.com/go/storage"
	"context"
	"fmt"
	"os"

	"io"
)

type Novel struct {
	ID            string
	Title         string
	Author        string
	PublishedDate string
	ImageURL      string
	Description   string
}

type NovelDatabase interface {
	ListNovels(context.Context) ([]*Novel, error)
	GetNovel(ctx context.Context, id string) (*Novel, error)
	AddNovel(ctx context.Context, n *Novel) (id string, err error)
	DeleteNovel(ctx context.Context, id string) error
	UpdateNovel(ctx context.Context, n *Novel) error
}

type Novelshelf struct {
	DB                NovelDatabase
	StorageBucket     *storage.BucketHandle
	StorageBucketName string
	logWriter         io.Writer
	errorClient       *errorreporting.Client
}

func NewNovelshelf(projectID string, db NovelDatabase) (*Novelshelf, error) {
	ctx := context.Background()

	bucketName := projectID + ".appspot.com"
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return nil, err
	}
	errorClient, err := errorreporting.NewClient(ctx, projectID, errorreporting.Config{
		ServiceVersion: "novelshelf",
		OnError: func(err error) {
			fmt.Fprintf(os.Stderr, "could not log error: %v", err)
		},
	})
	if err != nil {
		return nil, fmt.Errorf("errorreporting.NewClient: %v", err)
	}

	n := &Novelshelf{
		DB:                db,
		StorageBucket:     storageClient.Bucket(bucketName),
		StorageBucketName: bucketName,
		logWriter:         os.Stderr,
		errorClient:       errorClient,
	}
	return n, nil
}
