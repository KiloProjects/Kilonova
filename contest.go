package kilonova

import "time"

type Contest struct {
	ID        int          `json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	Name      string       `json:"name"`
	Editors   []*UserBrief `json:"editors"`
	Testers   []*UserBrief `json:"tester"`

	// PublicJoin indicates wether a user can freely join a contest
	// or he needs to be manually added
	PublicJoin bool `json:"public_join"`

	// Hidden indicates wether a contest is hidden from others
	// Contestants may be able to see the contest
	Hidden bool `json:"hidden"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// MaxSubs is the maximum number of submissions
	// that someone is allowed to send to a problem during a contest
	// < 0 => no limit
	MaxSubs int `json:"max_subs"`
}

type ContestUpdate struct {
	Name *string `json:"name"`

	PublicJoin *bool `json:"public_join"`
	Hidden     *bool `json:"hidden"`

	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`

	MaxSubs *int `json:"max_subs"`
}

type ContestQuestion struct {
	AuthorID  int       `json:"author_id"`
	AskedAt   time.Time `json:"asked_at"`
	ContestID int       `json:"contest_id"`
	Text      string    `json:"text"`

	// Maybe?
	// (?) ProblemID int `json:"problem_id"`

	ResponseAuthorID int       `json:"response_author_id"`
	ResponedAt       time.Time `json:"responded_at"`
	Response         string    `json:"response"`
}

type ContestAnnouncement struct {
	CreatedAt   time.Time `json:"created_at"`
	AnnouncerID int       `json:"announcer_id"`
	Text        string    `json:"text"`
}

// Maybe? Not sure if to actually make it
// But define it nevertheless
type ContestRegistration struct {
	CreatedAt time.Time `json:"created_at"`
	ContestID int       `json:"contest_id"`
	UserID    int       `json:"user_id"`
}
