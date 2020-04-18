package domain

type Novel struct {
	ID            int64
	Title         string
	Author        string
	PublishedDate string
	ImageURL      string
	Description   string
	CreatedBy     string
	CreatedByID   string
}

func (n *Novel) SetCreatorAnonymous() {
	n.CreatedBy = ""
	n.CreatedByID = "anonymous"
}

type NovelDatabase interface {
	ListNovels() ([]*Novel, error)
	ListNovelsCreatedBy(userID string) ([]*Novel, error)
	GetNovel(id int64) (*Novel, error)
	AddNovel(n *Novel) (id int64, err error)
	DeleteNovel(id int64) error
	UpdateBook(n *Novel) error
	Close()
}
