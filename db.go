package kilonova

import (
	"context"
	"io"
)

type DB interface {

	// UserService
	User(ctx context.Context, id int) (*User, error)
	Users(ctx context.Context, filter UserFilter) ([]*User, error)
	CountUsers(ctx context.Context, filter UserFilter) (int, error)
	UserExists(ctx context.Context, username, email string) (bool, error)

	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, id int, upd UserUpdate) error
	DeleteUser(ctx context.Context, id int) error

	// ProblemService
	Problem(ctx context.Context, id int) (*Problem, error)
	Problems(ctx context.Context, filter ProblemFilter) ([]*Problem, error)

	CreateProblem(ctx context.Context, problem *Problem) error
	UpdateProblem(ctx context.Context, id int, upd ProblemUpdate) error
	BulkUpdateProblems(ctx context.Context, filter ProblemFilter, upd ProblemUpdate) error
	DeleteProblem(ctx context.Context, id int) error

	// TestService
	CreateTest(ctx context.Context, test *Test) error

	Test(ctx context.Context, problemID, testVID int) (*Test, error)
	TestByID(ctx context.Context, id int) (*Test, error)
	Tests(ctx context.Context, problemID int) ([]*Test, error)

	UpdateTest(ctx context.Context, id int, upd TestUpdate) error

	// We don't delete tests, we orphan them
	// DeleteTest(ctx context.Context, id int) error

	OrphanProblemTests(ctx context.Context, problemID int) error
	OrphanProblemTest(ctx context.Context, problemID int, testVID int) error
	BiggestVID(ctx context.Context, problemID int) (int, error)

	// SubTaskService
	CreateSubTask(ctx context.Context, stask *SubTask) error

	SubTask(ctx context.Context, pbid, stvid int) (*SubTask, error)
	SubTaskByID(ctx context.Context, stid int) (*SubTask, error)
	SubTasksByTest(ctx context.Context, problemid, testid int) ([]*SubTask, error)
	SubTasks(ctx context.Context, pbid int) ([]*SubTask, error)

	UpdateSubTask(ctx context.Context, id int, upd SubTaskUpdate) error

	DeleteSubTask(ctx context.Context, stid int) error
	DeleteSubTasks(ctx context.Context, pbid int) error

	// SubmissionService
	Submission(ctx context.Context, id int) (*Submission, error)

	Submissions(ctx context.Context, filter SubmissionFilter) ([]*Submission, error)
	CountSubmissions(ctx context.Context, filter SubmissionFilter) (int, error)

	CreateSubmission(ctx context.Context, sub *Submission) error
	UpdateSubmission(ctx context.Context, id int, upd SubmissionUpdate) error

	BulkUpdateSubmissions(ctx context.Context, filter SubmissionFilter, upd SubmissionUpdate) error
	DeleteSubmission(ctx context.Context, id int) error

	MaxScore(ctx context.Context, userid, problemid int) int
	MaxScores(ctx context.Context, userid int, problemids []int) map[int]int
	SolvedProblems(ctx context.Context, userid int) ([]int, error)

	// SubTestService
	SubTestsBySubID(ctx context.Context, subid int) ([]*SubTest, error)
	SubTest(ctx context.Context, id int) (*SubTest, error)

	CreateSubTest(ctx context.Context, subtest *SubTest) error
	UpdateSubTest(ctx context.Context, id int, upd SubTestUpdate) error
	UpdateSubmissionSubTests(ctx context.Context, subID int, upd SubTestUpdate) error

	// ProblemListService
	ProblemList(ctx context.Context, id int) (*ProblemList, error)
	ProblemLists(ctx context.Context, filter ProblemListFilter) ([]*ProblemList, error)

	CreateProblemList(ctx context.Context, pblist *ProblemList) error
	UpdateProblemList(ctx context.Context, id int, upd ProblemListUpdate) error
	DeleteProblemList(ctx context.Context, id int) error

	// AttachmentService
	CreateAttachment(context.Context, *Attachment) error
	Attachment(ctx context.Context, id int) (*Attachment, error)
	Attachments(ctx context.Context, getData bool, filter AttachmentFilter) ([]*Attachment, error)
	UpdateAttachment(ctx context.Context, id int, upd AttachmentUpdate) error
	DeleteAttachment(ctx context.Context, attid int) error
	DeleteAttachments(ctx context.Context, pbid int) error

	// Sessioner
	CreateSession(ctx context.Context, uid int) (string, error)
	GetSession(ctx context.Context, sess string) (int, error)
	RemoveSession(ctx context.Context, sess string) error

	// Verificationer
	CreateVerification(ctx context.Context, id int) (string, error)
	GetVerification(ctx context.Context, verif string) (int, error)
	RemoveVerification(ctx context.Context, verif string) error

	io.Closer
}
