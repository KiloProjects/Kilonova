package kilonova

import "time"

// Just a sketch of the concepts behind a blog functionality
type BlogPost struct {
	ID        int
	CreatedAt time.Time
	AuthorID  int

	ParentID *int // If it's a reply

	Slug     string // unique, used in URL
	Visible  bool   // Valid only if parentID is nil(?)
	Featured bool   // Featured on a special "list" on the home page
}
type BlogPostAttachment struct{}
type UserFollow struct{}
