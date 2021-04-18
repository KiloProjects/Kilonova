package kilonova

import (
	"context"
	"time"
)

type ProblemList struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	AuthorID    int       `json:"author_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	List        []int     `json:"list"`
}

type ProblemListFilter struct {
	ID       *int `json:"id"`
	AuthorID *int `json:"author_id"`
	// List     []int `json:"list"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ProblemListUpdate struct {
	AuthorID    *int    `json:"author_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	List        []int   `json:"list"`
}

type ProblemListService interface {
	ProblemList(ctx context.Context, id int) (*ProblemList, error)
	ProblemLists(ctx context.Context, filter ProblemListFilter) ([]*ProblemList, error)

	CreateProblemList(ctx context.Context, pblist *ProblemList) error
	UpdateProblemList(ctx context.Context, id int, upd ProblemListUpdate) error
	DeleteProblemList(ctx context.Context, id int) error
}
