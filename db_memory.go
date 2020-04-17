package main

import (
	"fmt"
	"sort"
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

func (m *memoryDB) ListNovels() ([]*Novel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var novels []*Novel
	for _, b := range m.novels {
		novels = append(novels, b)
	}
	return novels, nil
}

func (m *memoryDB) ListNovelsCreatedBy(userID string) ([]*Novel, error) {
	if userID == "" {
		return m.ListNovels()
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	var novels []*Novel
	for _, b := range m.novels {
		if b.CreatedByID == userID {
			novels = append(novels, b)
		}
	}
	sort.Sort(novelByTitle(novels))
	return novels, nil
}

func (m *memoryDB) GetNovel(id int64) (*Novel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	novel, ok := m.novels[id]
	if !ok {
		return nil, fmt.Errorf("memorydb: book not found with ID %d", id)
	}
	return novel, nil
}

func (m *memoryDB) AddNovel(b *Novel) (id int64, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	b.ID = m.nextID
	m.novels[b.ID] = b
	m.nextID++

	return m.nextID, nil
}

func (m *memoryDB) DeleteNovel(id int64) error {
	if id == 0 {
		return fmt.Errorf("memorydb: book with unassigned ID passed into deleteBook")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.novels[id]; !ok {
		return fmt.Errorf("memorydb: book with unassigned ID passed with ID %d, does not exist", id)
	}
	delete(m.novels, id)
	return nil
}

func (m *memoryDB) UpdateBook(b *Novel) error {
	if b.ID == 0 {
		return fmt.Errorf("memorydb: book with unassigned ID passed into updateBook")
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.novels[b.ID] = b
	return nil
}

func (m *memoryDB) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.novels = nil
}

type novelByTitle []*Novel

func (s novelByTitle) Less(i, j int) bool { return s[i].Title < s[j].Title }
func (s novelByTitle) Len() int           { return len(s) }
func (s novelByTitle) Swap(i, j int)      { s[i], s[j] = s[j], s[i] }
