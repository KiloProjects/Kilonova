package kilonova

import (
	"log/slog"
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

	SidebarHidable    bool `json:"sidebar_hidable"`
	FeaturedChecklist bool `json:"featured_checklist"`

	// This is a separate type and not a ProblemList because it might
	// not necessairly be a tree-like structure (ie. it might have cycles)
	SubLists []*ShallowProblemList `json:"sublists"`
}

func (p *ProblemList) LogValue() slog.Value {
	return slog.GroupValue(slog.Int("id", p.ID), slog.String("name", p.Title))
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

	SidebarHidable    bool `json:"sidebar_hidable"`
	FeaturedChecklist bool `json:"featured_checklist"`
	// NumProblems holds the number of problems including sublists
	NumProblems int `json:"num_problems"`
}

type ProblemListUpdate struct {
	AuthorID    *int    `json:"author_id"`
	Title       *string `json:"title"`
	Description *string `json:"description"`

	SidebarHidable    *bool `json:"sidebar_hidable"`
	FeaturedChecklist *bool `json:"featured_checklist"`
}

type ProblemListFilter struct {
	Root              bool  `json:"root"`
	FeaturedChecklist *bool `json:"featured_checklist"`
	// Note that results with ParentID include that parent as well, it should print out the entire tree
	ParentID *int `json:"parent_id"`
}
