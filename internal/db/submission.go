package db

import (
	"context"
	"database/sql"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
)

type Status string

const (
	StatusWaiting  Status = "waiting"
	StatusWorking  Status = "working"
	StatusFinished Status = "finished"
)

type Submission struct {
	ctx   context.Context
	db    *DB
	Empty bool `json:"-"`

	ID             int64          `json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UserID         int64          `json:"user_id"`
	ProblemID      int64          `json:"problem_id"`
	Language       string         `json:"language"`
	Code           string         `json:"code"`
	Status         Status         `json:"status"`
	CompileError   sql.NullBool   `json:"compile_error"`
	CompileMessage sql.NullString `json:"compile_message"`
	Score          int32          `json:"score"`
	Visible        bool           `json:"visible"`
}

type SubTest struct {
	ctx context.Context
	db  *DB

	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Done         bool      `json:"done"`
	Verdict      string    `json:"verdict"`
	Time         float64   `json:"time,omitempty"`
	Memory       int32     `json:"memory,omitempty"`
	Score        int32     `json:"score,omitempty"`
	TestID       int64     `json:"test_id"`
	UserID       int64     `json:"user_id"`
	SubmissionID int64     `json:"submission_id"`
}

func (s *SubTest) SetData(memory int32, score int, time float64, verdict string) error {
	return s.db.raw.SetSubmissionTest(s.ctx, rawdb.SetSubmissionTestParams{ID: s.ID, Memory: memory, Score: int32(score), Time: time, Verdict: verdict})
}

func (s *Submission) SetStatus(status Status, score int) error {
	return s.db.raw.SetSubmissionStatus(s.ctx, rawdb.SetSubmissionStatusParams{ID: s.ID, Status: rawdb.Status(status), Score: int32(score)})
}

func (s *Submission) SetCompilation(cError bool, cMsg string) error {
	return s.db.raw.SetCompilation(s.ctx, rawdb.SetCompilationParams{
		ID:             s.ID,
		CompileError:   sql.NullBool{Bool: cError, Valid: true},
		CompileMessage: sql.NullString{String: cMsg, Valid: true},
	})
}

func (s *Submission) SetVisibility(visible bool) error {
	if err := s.db.raw.SetSubmissionVisibility(s.ctx, rawdb.SetSubmissionVisibilityParams{ID: s.ID, Visible: visible}); err != nil {
		return err
	}
	s.Visible = visible
	return nil
}

//////////////////////////////////////////////////

func (db *DB) Submission(ctx context.Context, id int64) (*Submission, error) {
	sub, err := db.raw.Submission(ctx, id)
	if err != nil {
		return nil, err
	}

	return db.subFromRaw(ctx, sub), nil
}

func (db *DB) Submissions(ctx context.Context) ([]*Submission, error) {
	subs, err := db.raw.Submissions(ctx)
	if err != nil {
		return nil, err
	}

	var submissions []*Submission
	for _, sub := range subs {
		submissions = append(submissions, db.subFromRaw(ctx, sub))
	}
	return submissions, nil
}

func (db *DB) WaitingSubmissions(ctx context.Context) ([]*Submission, error) {
	subs, err := db.raw.WaitingSubmissions(ctx)
	if err != nil {
		return nil, err
	}

	var submissions []*Submission
	for _, sub := range subs {
		submissions = append(submissions, db.subFromRaw(ctx, sub))
	}
	return submissions, nil
}

func (db *DB) SubTest(ctx context.Context, id int64) (*SubTest, error) {
	subTest, err := db.raw.SubTest(ctx, id)
	if err != nil {
		return nil, err
	}

	return db.subTestFromRaw(ctx, subTest), nil
}

func (db *DB) SubTests(ctx context.Context, subID int64) ([]*SubTest, error) {
	subTests, err := db.raw.SubTests(ctx, subID)
	if err != nil {
		return nil, err
	}

	return db.subTestsFromRaw(ctx, subTests), nil
}

//////////////////////////////////////////////////

func (db *DB) AddSubTest(ctx context.Context, userID int64, testID int64, subID int64) error {
	return db.raw.CreateSubTest(ctx, rawdb.CreateSubTestParams{UserID: userID, TestID: testID, SubmissionID: subID})
}

// AddSubmission adds the submission to the DB, but also creates the subtests
func (db *DB) AddSubmission(ctx context.Context, userID int64, problemID int64, code string, lang string, visible bool) (int64, error) {
	pb, err := db.Problem(ctx, problemID)
	if err != nil {
		return -1, err
	}

	tests, err := pb.Tests()
	if err != nil {
		return -1, err
	}

	// Add submission
	id, err := db.raw.CreateSubmission(ctx, rawdb.CreateSubmissionParams{
		UserID:    userID,
		ProblemID: problemID,
		Code:      code,
		Language:  lang,
		Visible:   visible,
	})
	if err != nil {
		return -1, err
	}

	// Add subtests
	for _, test := range tests {
		if err := db.raw.CreateSubTest(ctx, rawdb.CreateSubTestParams{UserID: userID, TestID: test.ID, SubmissionID: id}); err != nil {
			return id, err
		}
	}

	return id, nil
}

//////////////////////////////////////////////////

func (db *DB) subFromRaw(ctx context.Context, sub rawdb.Submission) *Submission {
	return &Submission{
		ctx: ctx,
		db:  db,

		ID:             sub.ID,
		CreatedAt:      sub.CreatedAt,
		UserID:         sub.UserID,
		ProblemID:      sub.ProblemID,
		Language:       sub.Language,
		Code:           sub.Code,
		Status:         Status(sub.Status),
		CompileError:   sub.CompileError,
		CompileMessage: sub.CompileMessage,
		Score:          sub.Score,
		Visible:        sub.Visible,
	}
}

func (db *DB) subTestFromRaw(ctx context.Context, subTest rawdb.SubmissionTest) *SubTest {
	return &SubTest{
		ctx: ctx,
		db:  db,

		ID:           subTest.ID,
		CreatedAt:    subTest.CreatedAt,
		Done:         subTest.Done,
		Verdict:      subTest.Verdict,
		Time:         subTest.Time,
		Memory:       subTest.Memory,
		Score:        subTest.Score,
		TestID:       subTest.TestID,
		UserID:       subTest.UserID,
		SubmissionID: subTest.SubmissionID,
	}
}

func (db *DB) subTestsFromRaw(ctx context.Context, subTests []rawdb.SubmissionTest) []*SubTest {
	var sTests []*SubTest
	for _, sTest := range subTests {
		sTests = append(sTests, db.subTestFromRaw(ctx, sTest))
	}

	return sTests
}
