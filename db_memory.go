package main

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
)

var _ NovelDatabase = &memoryDB{}

type memoryDB struct {
	mu     sync.Mutex
	nextID int64
	novels map[string]*Novel
}

func newMemoryDB() *memoryDB {
	return &memoryDB{
		novels: make(map[string]*Novel),
		nextID: 1,
	}
}

func (db *memoryDB) Close(ctx context.Context) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.novels = nil
	return nil
}

func (db *memoryDB) ListNovels(ctx context.Context) ([]*Novel, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	var novels []*Novel
	for _, n := range db.novels {
		novels = append(novels, n)
	}
	sort.Slice(novels, func(i, j int) bool {
		return novels[i].Title < novels[j].Title
	})
	return novels, nil
}

func (db *memoryDB) GetNovel(_ context.Context, id string) (*Novel, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	novel, ok := db.novels[id]
	if !ok {
		return nil, fmt.Errorf("memorydb: novel not found with ID %q", id)
	}
	return novel, nil
}

func (db *memoryDB) AddNovel(ctx context.Context, n *Novel) (id string, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	n.ID = strconv.FormatInt(db.nextID, 10)
	db.novels[n.ID] = n

	db.nextID++

	return n.ID, nil
}

func (db *memoryDB) DeleteNovel(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("memorydb: novel with unassigned ID passed into DeletenNovel")
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	if _, ok := db.novels[id]; !ok {
		return fmt.Errorf("memorydb: could not delete novel with ID %q, does not exist", id)
	}
	delete(db.novels, id)
	return nil
}

func (db *memoryDB) UpdateNovel(ctx context.Context, n *Novel) error {
	if n.ID == "" {
		return fmt.Errorf("memorydb: novel with unassigned ID pass into Update")
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	db.novels[n.ID] = n
	return nil
}
