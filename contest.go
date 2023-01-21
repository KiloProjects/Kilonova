package kilonova

import "time"

/*
	Contest parameters:
		- public join:
			- false: only manually added users can be registered before the contest starts;
			- true: registrations are open for anyone before the contest starts.
		- visible:
			- false: contest can't be seen on the main page neither before nor during or after it's held;
			- true: contest can be seen on the main page and its contents can be accessed while it's held by unregistered users.
*/

type Contest struct {
	ID        int          `json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	Name      string       `json:"name"`
	Editors   []*UserBrief `json:"editors"`
	Testers   []*UserBrief `json:"tester"`

	// PublicJoin indicates whether a user can freely join a contest
	// or he needs to be manually added
	PublicJoin bool `json:"public_join"`

	// Visible indicates whether a contest can be seen by others
	// Contestants may be able to see the contest
	Visible bool `json:"hidden"`

	// Virtual indicates a virtual contest.
	// TODO: Implement this
	// Virtual bool `json:"virtual"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// MaxSubs is the maximum number of submissions
	// that someone is allowed to send to a problem during a contest
	// < 0 => no limit
	MaxSubs int `json:"max_subs"`
}

func (c *Contest) Started() bool {
	if c == nil {
		return false
	}
	return c.StartTime.Before(time.Now())
}

func (c *Contest) Ended() bool {
	if c == nil {
		return false
	}
	return c.EndTime.Before(time.Now())
}

func (c *Contest) Running() bool {
	if c == nil {
		return false
	}
	return c.Started() && !c.Ended()
}

type ContestUpdate struct {
	Name *string `json:"name"`

	PublicJoin *bool `json:"public_join"`
	Visible    *bool `json:"visible"`

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
