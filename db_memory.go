package main

import (
	"fmt"
	"sync"
)

var _ NovelDatabase = &memoryDB{}

type memoryDB struct {
	mu     sync.Mutex
	nextID int64
	novels map[int64]*Novel
}

func newMemoryDB() *memoryDB {
	return &memoryDB{
		nextID: 1,
		novels: make(map[int64]*Novel),
	}
}

func (m memoryDB) ListNovels() ([]*Novel, error) {
	panic("implement me")
}

func (m memoryDB) ListNovelsCreatedBy(userID string) ([]*Novel, error) {
	panic("implement me")
}

func (m memoryDB) GetNovel(id int64) (*Novel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	novel, ok := m.novels[id]
	if !ok {
		return nil, fmt.Errorf("memorydb: book not found with ID %d", id)
	}
	return novel, nil
}

func (m memoryDB) AddNovel(b *Novel) (id int64, err error) {
	panic("implement me")
}

func (m memoryDB) DeleteNovel(id int64) error {
	panic("implement me")
}

func (m memoryDB) UpdateBook(b *Novel) error {
	panic("implement me")
}

func (m memoryDB) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.novels = nil
}
