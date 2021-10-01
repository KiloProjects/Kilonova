package kilonova

import (
	"time"
)

type ProblemList struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	AuthorID    int       `json:"author_id" db:"author_id"`
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
