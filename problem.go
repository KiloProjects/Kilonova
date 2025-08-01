package kilonova

import (
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/shopspring/decimal"
)

var (
	DefaultSourceSize   = config.GenFlag[int]("behavior.problem.default_source_size", 30000, "Default maximum source code size for problems")
	ErrAttachmentExists = Statusf(400, "Attachment with that name already exists!")
)

type ScoringType string

const (
	ScoringTypeNone        ScoringType = ""
	ScoringTypeMaxSub      ScoringType = "max_submission"
	ScoringTypeSumSubtasks ScoringType = "sum_subtasks"
	ScoringTypeICPC        ScoringType = "acm-icpc"
)

type TaskType string

const (
	TaskTypeNone          TaskType = ""
	TaskTypeBatch         TaskType = "batch"
	TaskTypeCommunication TaskType = "communication"
)

type Problem struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
	TestName  string    `json:"test_name"`

	DefaultPoints decimal.Decimal `json:"default_points"`

	Visible      bool `json:"visible"`
	VisibleTests bool `json:"visible_tests"`

	// Limit stuff
	TimeLimit   float64 `json:"time_limit"`
	MemoryLimit int     `json:"memory_limit"`
	SourceSize  int     `json:"source_size"`

	SourceCredits string `json:"source_credits"`

	// Used only for leaderboard scoring right now
	ScoreScale decimal.Decimal `json:"score_scale"`

	// Eval stuff
	ConsoleInput   bool  `json:"console_input"`
	ScorePrecision int32 `json:"score_precision"`

	PublishedAt     *time.Time  `json:"published_at"`
	ScoringStrategy ScoringType `json:"scoring_strategy"`

	TaskType TaskType `json:"task_type"`

	// CommunicationProcesses is the number of processes that will be run in parallel for communication tasks
	CommunicationProcesses int `json:"communication_processes"`
}

func (pb *Problem) LogValue() slog.Value {
	if pb == nil {
		return slog.Value{}
	}
	return slog.GroupValue(slog.Int("id", pb.ID), slog.String("name", pb.Name))
}

type StatementVariant struct {
	// Language, ie. ro/en
	Language string `json:"lang"`
	// Format, ie. pdf/md/etc.
	Format string `json:"format"`
	// Type, ie. normal/short/llm/etc.
	Type string `json:"type"`
	// Private is true if the attachment for this statement variant is private.
	// it may be private if it's currently being worked on.
	Private bool `json:"public"`

	LastUpdatedAt time.Time `json:"last_updated_at"`
}

// Used for comparing in templates if the right option is selected.
func (sv *StatementVariant) Equals(other *StatementVariant) bool {
	return sv.Language == other.Language && sv.Format == other.Format && sv.Type == other.Type
}

type ScoredProblem struct {
	Problem
	ScoreUserID *int `json:"score_user_id"`

	MaxScore *decimal.Decimal `json:"max_score"`
	// For showing the published/unpublished label on front page
	IsEditor bool `json:"is_editor"`
}

// ProblemFilter is the struct with all filterable fields on the problem
// It also provides a Limit and Offset field, for pagination
// This list might be expanded as time goes on
type ProblemFilter struct {
	ID           *int    `json:"id"`
	IDs          []int   `json:"ids"`
	ConsoleInput *bool   `json:"console_input"`
	Visible      *bool   `json:"visible"`
	Name         *string `json:"name"`

	FuzzyName *string `json:"name_fuzzy"`

	// DeepListID - the list ID in which to search recursively for problems
	DeepListID *int `json:"deep_list_id"`

	// EditorUserID filter marks if the user is part of the *editors* of the problem
	// Note that it excludes factors like admin or contest editor, it's just the editors in the access section.
	EditorUserID *int `json:"editor_user_id"`

	Tags []*TagGroup `json:"tags"`

	Look             bool       `json:"-"`
	LookEditor       bool       `json:"-"`
	LookFullyVisible bool       `json:"-"`
	LookingUser      *UserBrief `json:"-"`

	// Should be "en" or "ro", if non-nil
	Language *string `json:"lang"`

	// Check problems that have attachment with that ID
	// Currently used for logging statement changes
	AttachmentID *int `json:"-"`

	// Used for getting problems for MOSS
	ContestID *int `json:"-"`

	UnsolvedBy  *int `json:"unsolved_by"`
	SolvedBy    *int `json:"solved_by"`
	AttemptedBy *int `json:"attempted_by"`

	// Unassociated filter ensures that all returned problems are not "bound" to a problem list
	Unassociated bool `json:"-"`

	// This is actually not used during filtering in DB, it's used by (*api.API).searchProblems
	ScoreUserID *int `json:"score_user_id"`

	Limit  uint64 `json:"limit"`
	Offset uint64 `json:"offset"`

	Ordering   string `json:"ordering"`
	Descending bool   `json:"descending"`
}

type ProblemUpdate struct {
	Name     *string `json:"name"`
	TestName *string `json:"test_name"`

	DefaultPoints *decimal.Decimal `json:"default_points"`

	ScoreScale *decimal.Decimal `json:"score_scale"`

	TimeLimit   *float64 `json:"time_limit"`
	MemoryLimit *int     `json:"memory_limit"`
	SourceSize  *int     `json:"source_size"`

	SourceCredits *string `json:"source_credits"`

	ConsoleInput *bool `json:"console_input"`
	Visible      *bool `json:"visible"`
	VisibleTests *bool `json:"visible_tests"`

	ScorePrecision  *int32      `json:"score_precision"`
	ScoringStrategy ScoringType `json:"scoring_strategy"`

	TaskType TaskType `json:"task_type"`

	CommunicationProcesses *int `json:"communication_processes"`
}

type Attachment struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Visible   bool      `json:"visible"`
	Private   bool      `json:"private"`
	Exec      bool      `json:"exec"`

	LastUpdatedAt time.Time `json:"last_updated_at"`
	LastUpdatedBy *int      `json:"last_updated_by"`

	Name string `json:"name"`
	// Data []byte `json:"data,omitempty"`
	Size int `json:"data_size"`
}

// Should be used only for interacting with db from sudoapi
type AttachmentFilter struct {
	ID         *int
	IDs        []int
	ProblemID  *int
	BlogPostID *int

	Visible *bool
	Private *bool
	Exec    *bool
	Name    *string

	Limit  uint64
	Offset uint64
}

type AttachmentUpdate struct {
	Visible *bool   `json:"visible"`
	Private *bool   `json:"private"`
	Exec    *bool   `json:"exec"`
	Name    *string `json:"name"`
}

type ProblemEvalSettings struct {
	// Files to be included during compilation, but not in the compile command
	HeaderFiles []string `json:"header_files"`
	// List of all grader files detected in attachments. Further processing is required to filter by language on evaluation
	GraderFiles []string `json:"grader_files"`
	// If the problem has a custom checker, this is non-empty.
	// If the problem is of type Communication, CheckerName will have "manager" as stem instead of "checker"
	CheckerName string `json:"has_checker"`
	// If the problem has a custom checker marked as legacy
	LegacyChecker bool `json:"legacy_checker"`

	// Stores the list of languages that are allowed to be submitted based on existing attachments
	LanguageWhitelist []string `json:"lang_whitelist"`
}

type ProblemChecklist struct {
	ProblemID        int  `json:"problem_id" db:"problem_id"`
	HasSourceCredits bool `json:"has_source_credits" db:"has_source"`

	NumPDF      int `json:"num_pdf_files" db:"num_pdf"`
	NumMarkdown int `json:"num_md_files" db:"num_md"`

	NumTests    int `json:"num_tests" db:"num_tests"`
	NumSubtasks int `json:"num_subtasks" db:"num_subtasks"`

	NumAuthorTags int `json:"num_author_tags" db:"num_authors"`
	NumOtherTags  int `json:"num_other_tags" db:"num_other_tags"`

	NumSolutions int `json:"num_sols" db:"num_sols"`
}

type ResourceType string

const (
	ResourceTypeNone      ResourceType = ""
	ResourceTypeEditorial ResourceType = "editorial"
	ResourceTypeVideo     ResourceType = "video"
	ResourceTypeSolution  ResourceType = "solution"
	ResourceTypeArticle   ResourceType = "article"
	ResourceTypeOther     ResourceType = "other"
)

type ExternalResource struct {
	ID          int       `json:"id" db:"id"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	URL         string    `json:"url" db:"url"`
	Language    string    `json:"language" db:"language"`
	Visible     bool      `json:"visible" db:"visible"`
	Accepted    bool      `json:"accepted" db:"accepted"`

	ProposedBy *int         `json:"proposed_by" db:"proposed_by"`
	Type       ResourceType `json:"type" db:"type"`
	ProblemID  int          `json:"problem_id" db:"problem_id"`

	Position int `json:"position" db:"position"`
}

func (r *ExternalResource) LogValue() slog.Value {
	if r == nil {
		return slog.Value{}
	}
	return slog.GroupValue(slog.String("name", r.Name), slog.Any("type", r.Type), slog.Int("pbid", r.ProblemID))
}

type ExternalResourceFilter struct {
	ID        *int         `json:"id"`
	ProblemID *int         `json:"problem_id"`
	Type      ResourceType `json:"type"`

	Language   *string `json:"language"`
	Visible    *bool   `json:"visible"`
	Accepted   *bool   `json:"accepted"`
	ProposedBy *int    `json:"proposed_by"`

	Look        bool       `json:"-"`
	LookingUser *UserBrief `json:"-"`

	Ordering   string `json:"ordering"`
	Descending bool   `json:"descending"`

	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type ExternalResourceUpdate struct {
	Name        *string      `json:"name"`
	Description *string      `json:"description"`
	URL         *string      `json:"url"`
	Language    *string      `json:"language"`
	Visible     *bool        `json:"visible"`
	Accepted    *bool        `json:"accepted"`
	Type        ResourceType `json:"type"`
	Position    *int         `json:"position"`
}

func (t ResourceType) FontAwesomeIcon() string {
	switch t {
	case ResourceTypeEditorial:
		return "fa-book"
	case ResourceTypeVideo:
		return "fa-film"
	case ResourceTypeSolution:
		return "fa-file-circle-check"
	case ResourceTypeArticle:
		return "fa-newspaper"
	case ResourceTypeOther:
		return "fa-file-circle-question"
	default:
		return "fa-file-lines"
	}
}
