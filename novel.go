package main

type Novel struct {
	ID            int64
	Title         int64
	Author        string
	PublishedDate string
	ImageURL      string
	Description   string
	CreatedBy     string
	CreatedByID   string
}

type NovelDatabase interface {
	ListNovels() ([]*Novel, error)
	ListNovelsCreatedBy(userID string) ([]*Novel, error)
	GetNovel(id int64) (*Novel, error)
	AddNovel(b *Novel) (id int64, err error)
	DeleteNovel(id int64) error
	UpdateBook(b *Novel) error
	Close()
}
