package kilonova

import (
	"log/slog"
	"net/netip"
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

type ContestType string

const (
	ContestTypeNone     ContestType = ""
	ContestTypeOfficial ContestType = "official"
	ContestTypeVirtual  ContestType = "virtual"
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

	// PublicLeaderboard controls whether the contest's leaderboard
	// is viewable by everybody or just admins
	PublicLeaderboard bool `json:"public_leaderboard"`

	LeaderboardStyle      LeaderboardType `json:"leaderboard_style"`
	LeaderboardFreeze     *time.Time      `json:"leaderboard_freeze"`
	ICPCSubmissionPenalty int             `json:"icpc_submission_penalty"`

	LeaderboardAdvancedFilter bool `json:"leaderboard_advanced_filter"`

	SubmissionCooldown time.Duration `json:"submission_cooldown"`
	QuestionCooldown   time.Duration `json:"question_cooldown"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// PerUserTime records the number of seconds a user has in an USACO-style participation
	// Setting it to 0 will make contests behave "normally"
	PerUserTime int `json:"per_user_time"`

	Type ContestType `json:"type"`

	// MaxSubs is the maximum number of submissions
	// that someone is allowed to send to a problem during a contest.
	// Any number < 0 means no limit
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

func (c *Contest) IsEditor(user *UserBrief) bool {
	if c == nil {
		return false
	}
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
		return true
	}

	for _, editor := range c.Editors {
		if editor.ID == user.ID {
			return true
		}
	}
	return false
}

// Tester = Testers + Editors + Admins
func (c *Contest) IsTester(user *UserBrief) bool {
	if c == nil {
		return false
	}
	if !user.IsAuthed() {
		return false
	}
	if user.IsAdmin() {
		return true
	}

	for _, editor := range c.Editors {
		if editor.ID == user.ID {
			return true
		}
	}
	for _, tester := range c.Testers {
		if tester.ID == user.ID {
			return true
		}
	}
	return false
}

func (c *Contest) LogValue() slog.Value {
	if c == nil {
		return slog.Value{}
	}
	return slog.GroupValue(slog.Int("id", c.ID), slog.String("name", c.Name))
}

type ContestFilter struct {
	ID          *int       `json:"id"`
	IDs         []int      `json:"ids"`
	Look        bool       `json:"-"`
	LookingUser *UserBrief `json:"-"`

	ProblemID *int `json:"problem_id"`

	// Shows contests in which user with this ID was registered
	ContestantID *int `json:"contestant_id"`

	// Shows contests in which user with this ID is an editor or not.
	// If NotEditor is true, then the contests are filtered such that the user ID is NOT an editor
	EditorID *int `json:"editor_id"`
	// NotEditor negates EditorID's behaviour
	NotEditor bool `json:"not_editor"`

	Future  bool `json:"future"`
	Running bool `json:"running"`
	Ended   bool `json:"ended"`

	Type ContestType `json:"type"`

	// Filters for that user the *important* contests:
	//   - Official contests
	//   - Virtual contests with participation
	//   - Virtual contests the user organizes (editor/tester)
	// This is used in filtering the contests for the main page
	ImportantContestsUID *int `json:"important_contest_uid"`

	Since *time.Time `json:"-"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`

	Ordering  string `json:"ordering"`
	Ascending bool   `json:"ascending"`
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

	LeaderboardAdvancedFilter *bool `json:"leaderboard_advanced_filter"`

	ChangeLeaderboardFreeze bool       `json:"change_leaderboard_freeze"`
	LeaderboardFreeze       *time.Time `json:"leaderboard_freeze"`

	// Normally I'd put a *time.Duration directly here, but schema has a hard time decoding them
	// So the convention is: the unit is an integer number of milliseconds (the resolution set in the DB)
	SubmissionCooldown *int `json:"submission_cooldown"`
	QuestionCooldown   *int `json:"question_cooldown"`

	Type ContestType `json:"type"`

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

type MOSSSubmission struct {
	ID        int       `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`

	ContestID int `json:"contest_id" db:"contest_id"`
	ProblemID int `json:"problem_id" db:"problem_id"`

	Language string `json:"language" db:"language"`

	URL      string `json:"url" db:"url"`
	SubCount int    `json:"subcount" db:"subcount"`
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

	RedeemCount int  `json:"redeem_count" db:"redeem_cnt"`
	MaxCount    *int `json:"max_invitation_count" db:"max_invitation_cnt"`

	Expired bool `json:"expired"`
}

func (ci *ContestInvitation) Invalid() bool {
	return ci.Expired || (ci.MaxCount != nil && *ci.MaxCount <= ci.RedeemCount)
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

	AdvancedFilter bool `json:"advanced_filter"`

	FreezeTime *time.Time      `json:"freeze_time"`
	Type       LeaderboardType `json:"type"`
}

type ContestLimitConfig struct {
	ContestID              int  `json:"contest_id"`
	IPManagementEnabled    bool `json:"ip_management_enabled"`
	WhitelistingEnabled    bool `json:"whitelisting_enabled"`
	PastSubmissionsEnabled bool `json:"past_submissions_enabled"`
}

type ContestLimitConfigUpdate struct {
	IPManagementEnabled    *bool `json:"ip_management_enabled"`
	WhitelistingEnabled    *bool `json:"whitelisting_enabled"`
	PastSubmissionsEnabled *bool `json:"past_submissions_enabled"`
}

type ContestUserIPs struct {
	User *UserBrief
	IPs  []*netip.Addr
}
