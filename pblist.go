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

	// NumProblems holds the number of problems including sublists
	NumProblems int `json:"num_problems"`

	SidebarHidable bool `json:"sidebar_hidable"`

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

	SidebarHidable bool `json:"sidebar_hidable"`

	// NumProblems holds the number of problems including sublists
	NumProblems int `json:"num_problems"`
}

type ProblemListUpdate struct {
	AuthorID    *int    `json:"author_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`

	SidebarHidable *bool `json:"sidebar_hidable"`
}
