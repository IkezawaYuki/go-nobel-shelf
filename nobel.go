package main

type Nobel struct {
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
	ListBooks() ([]*Nobel, error)
}
