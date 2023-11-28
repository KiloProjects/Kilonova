package kilonova

import (
	"time"

	"github.com/shopspring/decimal"
)

/*
	Contest parameters:
		- public join:
			- false: only manually added users can be registered before the contest starts;
			- true: registrations are open for anyone before the contest starts.
		- visible:
			- false: contest can't be seen on the main page neither before nor during or after it's held;
			- true: contest can be seen on the main page and its contents can be accessed while it's held by unregistered users.
*/

type LeaderboardType string

const (
	LeaderboardTypeNone    LeaderboardType = ""
	LeaderboardTypeClassic LeaderboardType = "classic"
	LeaderboardTypeICPC    LeaderboardType = "acm-icpc"
)

type Contest struct {
	ID        int          `json:"id"`
	CreatedAt time.Time    `json:"created_at"`
	Name      string       `json:"name"`
	Editors   []*UserBrief `json:"editors"`
	Testers   []*UserBrief `json:"testers"`

	Description string `json:"description"`

	// PublicJoin indicates whether a user can freely join a contest
	// or he needs to be manually added
	PublicJoin bool `json:"public_join"`

	// RegisterDuringContest indicates whether a user can join a contest while it's running
	// It is useless without PublicJoin set to true
	RegisterDuringContest bool `json:"register_during_contest"`

	// Visible indicates whether a contest can be seen by others
	// Contestants may be able to see the contest
	Visible bool `json:"hidden"`

	// Virtual indicates a virtual contest.
	// TODO: Implement this
	// Virtual bool `json:"virtual"`

	// PublicLeaderboard controls whether the contest's leaderboard
	// is viewable by everybody or just admins
	PublicLeaderboard bool `json:"public_leaderboard"`

	LeaderboardStyle      LeaderboardType `json:"leaderboard_style"`
	LeaderboardFreeze     *time.Time      `json:"leaderboard_freeze"`
	ICPCSubmissionPenalty int             `json:"icpc_submission_penalty"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// PerUserTime records the number of seconds a user has in an USACO-style participation
	// Setting it to 0 will make contests behave "normally"
	PerUserTime int `json:"per_user_time"`

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

	Description *string `json:"description"`

	StartTime *time.Time `json:"start_time"`
	EndTime   *time.Time `json:"end_time"`

	MaxSubs *int `json:"max_subs"`

	RegisterDuringContest *bool `json:"register_during_contest"`

	PublicLeaderboard     *bool           `json:"public_leaderboard"`
	LeaderboardStyle      LeaderboardType `json:"leaderboard_style"`
	ICPCSubmissionPenalty *int            `json:"icpc_submission_penalty"`

	ChangeLeaderboardFreeze bool       `json:"change_leaderboard_freeze"`
	LeaderboardFreeze       *time.Time `json:"leaderboard_freeze"`

	PerUserTime *int `json:"per_user_time"` // Seconds
}

type ContestQuestion struct {
	ID        int       `json:"id"`
	AuthorID  int       `json:"author_id"`
	AskedAt   time.Time `json:"asked_at"`
	ContestID int       `json:"contest_id"`
	Text      string    `json:"text"`

	ResponedAt *time.Time `json:"responded_at"`
	Response   *string    `json:"response"`
}

type ContestAnnouncement struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ContestID int       `json:"contest_id"`
	Text      string    `json:"text"`
}

type ContestRegistration struct {
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ContestID int       `json:"contest_id" db:"contest_id"`
	UserID    int       `json:"user_id" db:"user_id"`

	IndividualStartTime *time.Time `json:"individual_start" db:"individual_start_at"`
	IndividualEndTime   *time.Time `json:"individual_end" db:"individual_end_at"`

	InvitationID *string `json:"invitation_id" db:"invitation_id"`
}

type ContestInvitation struct {
	ID string `json:"id"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
	ContestID int       `json:"contest_id" db:"contest_id"`
	CreatorID *int      `json:"creator_id" db:"creator_id"`

	Expired bool `json:"expired"`
}

// TODO: Maybe it would be nicer to coalesce all problem maps in a struct?
type LeaderboardEntry struct {
	User *UserBrief `json:"user"`

	// For classic mode
	ProblemScores map[int]decimal.Decimal `json:"scores"`
	TotalScore    decimal.Decimal         `json:"total"`

	// For ICPC mode
	ProblemAttempts map[int]int `json:"attempts"`
	Penalty         int         `json:"penalty"`
	NumSolved       int         `json:"num_solved"`
	// ProblemTimes is expressed as number of minutes since start
	ProblemTimes map[int]float64 `json:"last_times"`

	LastTime   *time.Time `json:"last_time"`
	FreezeTime *time.Time `json:"freeze_time"`
}

type ContestLeaderboard struct {
	ProblemOrder []int               `json:"problem_ordering"`
	ProblemNames map[int]string      `json:"problem_names"`
	Entries      []*LeaderboardEntry `json:"entries"`

	FreezeTime *time.Time      `json:"freeze_time"`
	Type       LeaderboardType `json:"type"`
}
