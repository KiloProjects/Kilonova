package db

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
	"github.com/jmoiron/sqlx"
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

	ID        int64     `json:"id" db:"id"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	User      *User     `json:"user,omitempty"`
	UserID    int64     `json:"user_id" db:"user_id"`
	Problem   *Problem  `json:"problem,omitempty"`
	ProblemID int64     `json:"problem_id" db:"problem_id"`
	Language  string    `json:"language"`
	Code      string    `json:"code,omitempty"`
	Status    Status    `json:"status"`

	CErrField      sql.NullBool   `json:"-" db:"compile_error"`
	CMsgField      sql.NullString `json:"-" db:"compile_message"`
	CompileError   bool           `json:"compile_error"`
	CompileMessage string         `json:"compile_message,omitempty"`

	Score    int32      `json:"score"`
	Visible  bool       `json:"visible"`
	SubTests []*SubTest `json:"sub_tests,omitempty"`
}

type SubTest struct {
	ctx context.Context
	db  *DB

	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Done         bool      `json:"done"`
	Verdict      string    `json:"verdict"`
	Time         float64   `json:"time"`
	Memory       int32     `json:"memory"`
	Score        int32     `json:"score"`
	Test         *Test     `json:"test,omitempty"`
	TestID       int64     `json:"test_id"`
	UserID       int64     `json:"user_id"`
	SubmissionID int64     `json:"submission_id"`
}

func (s *SubTest) GetTest() (*Test, error) {
	test, err := s.db.TestByID(s.ctx, s.TestID)
	if err != nil {
		return nil, err
	}
	s.Test = test
	return test, nil
}

func (s *SubTest) SetData(memory int32, score int, time float64, verdict string) error {
	return s.db.raw.SetSubmissionTest(s.ctx, rawdb.SetSubmissionTestParams{ID: s.ID, Memory: memory, Score: int32(score), Time: time, Verdict: verdict})
}

func (s *Submission) GetSubTests() ([]*SubTest, error) {
	subtests, err := s.db.SubTests(s.ctx, s.ID)
	if err != nil {
		return nil, err
	}

	for i := range subtests {
		if _, err := subtests[i].GetTest(); err != nil {
			log.Println(err)
			return nil, err
		}
	}

	s.SubTests = subtests
	return subtests, nil
}

func (s *Submission) GetProblem() (*Problem, error) {
	pb, err := s.db.Problem(s.ctx, s.ProblemID)
	if err != nil {
		return nil, err
	}

	s.Problem = pb
	return pb, nil
}

func (s *Submission) GetUser() (*User, error) {
	user, err := s.db.User(s.ctx, s.UserID)
	if err != nil {
		return nil, err
	}
	s.User = user
	return user, nil
}

func (s *Submission) LoadAll() error {
	if _, err := s.GetSubTests(); err != nil {
		return err
	}
	if _, err := s.GetProblem(); err != nil {
		return err
	}
	if _, err := s.GetUser(); err != nil {
		return err
	}
	return nil
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

type SubmissionFilter struct {
	UserID    int64
	ProblemID int64

	Status       Status
	Lang         string
	Visible      *bool
	Score        *int32
	CompileError *bool
}

func (s SubmissionFilter) getQuery(db *sqlx.DB) (string, []interface{}) {
	var data []interface{}
	q := "SELECT * FROM submissions WHERE 1 = 1"

	if s.UserID != 0 {
		q += " AND user_id = ?"
		data = append(data, s.UserID)
	}

	if s.ProblemID != 0 {
		q += " AND problem_id = ?"
		data = append(data, s.ProblemID)
	}

	if s.Status != "" {
		q += " AND status = ?"
		data = append(data, string(s.Status))
	}

	if s.Lang != "" {
		q += " AND language = ?"
		data = append(data, s.Lang)
	}

	if s.Visible != nil {
		q += " AND visible = ?"
		data = append(data, *s.Visible)
	}

	if s.Score != nil {
		q += " AND score = ?"
		data = append(data, *s.Score)
	}

	if s.CompileError != nil {
		q += " AND compile_error = ?"
		data = append(data, *s.CompileError)
	}

	q += " ORDER BY id DESC"

	// TODO (?): Time frames

	return db.Rebind(q), data
}

func (db *DB) FilterSubmissions(ctx context.Context, filter SubmissionFilter) ([]*Submission, error) {
	var subs []*Submission

	q, data := filter.getQuery(db.dbconn)

	if err := db.dbconn.SelectContext(ctx, &subs, q, data...); err != nil {
		return nil, err
	}

	for i := range subs {
		subs[i].ctx = ctx
		subs[i].db = db

		subs[i].CompileError = subs[i].CErrField.Bool
		subs[i].CompileMessage = subs[i].CMsgField.String
	}

	return subs, nil
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

func (db *DB) ResetWaitingStatus(ctx context.Context) error {
	return db.raw.ResetWaitingStatus(ctx)
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
		CompileError:   sub.CompileError.Bool,
		CompileMessage: sub.CompileMessage.String,
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
