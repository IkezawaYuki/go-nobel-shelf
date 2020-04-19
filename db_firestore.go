package main

import (
	"cloud.google.com/go/firestore"
	"context"
	"fmt"
	"google.golang.org/api/iterator"
	"log"
)

type firestoreDB struct {
	client *firestore.Client
}

var _ NovelDatabase = &firestoreDB{}

func newFirestoreDB(client *firestore.Client) (*firestoreDB, error) {
	ctx := context.Background()
	err := client.RunTransaction(ctx, func(ctx context.Context, t *firestore.Transaction) error {
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("firestoredb: could not connect: %v", err)
	}
	return &firestoreDB{client: client}, nil
}

func (db *firestoreDB) Close(ctx context.Context) error {
	return db.client.Close()
}

func (db *firestoreDB) ListNovels(ctx context.Context) ([]*Novel, error) {
	novels := make([]*Novel, 0)
	iter := db.client.Collection("novels").Query.OrderBy("Title", firestore.Asc).Documents(ctx)
	defer iter.Stop()
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("firestoredb: could not list novels: %v", err)
		}
		n := &Novel{}
		doc.DataTo(n)
		log.Printf("Novel %q ID: %q", n.Title, n.ID)
		novels = append(novels, n)
	}
	return novels, nil
}

func (db *firestoreDB) GetNovel(ctx context.Context, id string) (*Novel, error) {
	ds, err := db.client.Collection("novels").Doc(id).Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("firestoredb: Get: %v", err)
	}
	n := &Novel{}
	ds.DataTo(n)
	return n, nil
}

func (db *firestoreDB) AddNovel(ctx context.Context, n *Novel) (id string, err error) {
	ref := db.client.Collection("novels").NewDoc()
	n.ID = ref.ID
	if _, err := ref.Create(ctx, n); err != nil {
		return "", fmt.Errorf("create: %v", err)
	}
	return ref.ID, nil
}

func (db *firestoreDB) DeleteNovel(ctx context.Context, id string) error {
	if _, err := db.client.Collection("novels").Doc(id).Delete(ctx); err != nil {
		return fmt.Errorf("firestore: delete: %v", err)
	}
	return nil
}

func (db *firestoreDB) UpdateNovel(ctx context.Context, n *Novel) error {
	if _, err := db.client.Collection("novels").Doc(n.ID).Set(ctx, n); err != nil {
		return fmt.Errorf("firestore: set: %v", err)
	}
	return nil
}
