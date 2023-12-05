package kilonova

import "time"

// Just a sketch of the concepts behind a blog functionality
type BlogPost struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	AuthorID  int       `json:"author_id"`

	Title string `json:"title"`

	Slug    string `json:"slug"` // unique, used in URL
	Visible bool   `json:"visible"`

	PublishedAt *time.Time `json:"published_at"`
}

type BlogPostFilter struct {
	ID       *int  `json:"id"`
	IDs      []int `json:"ids"`
	AuthorID *int  `json:"author_id"`

	Slug *string `json:"slug"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	Look        bool       `json:"-"`
	LookingUser *UserBrief `json:"-"`

	// Check posts that have attachment with that ID
	// Currently used for logging statement changes
	AttachmentID *int `json:"-"`

	Ordering  string `json:"ordering"`
	Ascending bool   `json:"ascending"`
}

type BlogPostUpdate struct {
	Slug    *string `json:"slug"`
	Visible *bool   `json:"visible"`
	Title   *string `json:"title"`
}

// type UserFollow struct{}
