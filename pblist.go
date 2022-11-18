package kilonova

import (
	"time"
)

type ProblemList struct {
	ID          int       `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	AuthorID    int       `json:"author_id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	List        []int     `json:"list"`

	// This is a separate type and not a ProblemList because it might
	// not necessairly be a tree-like structure (ie. it might have cycles)
	SubLists []*ShallowProblemList `json:"sublists"`
}

func (p *ProblemList) ProblemIDs() []int {
	if p == nil {
		return nil
	}
	return p.List
}

type ShallowProblemList struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	AuthorID int    `json:"author_id"`

	// Should only be used to precompute the number of solved problems, might be deleted sometime else
	List []int `json:"list"`
}

type ProblemListFilter struct {
	ID       *int `json:"id"`
	AuthorID *int `json:"author_id"`
	// List     []int `json:"list"`
	Root bool `json:"root"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ProblemListUpdate struct {
	AuthorID    *int    `json:"author_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`
	//List        []int   `json:"list"`
}
