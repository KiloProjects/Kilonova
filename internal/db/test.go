package db

import (
	"context"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
)

type Test struct {
	ctx   context.Context
	db    *DB
	Empty bool `json:"-"`

	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Score     int32     `json:"score"`

	Problem   *Problem `json:"problem"`
	ProblemID int64    `json:"problem_id"`

	VisibleID int64 `json:"visible_id"`
	Orphaned  bool  `json:"orphaned"`
}

func (t *Test) GetProblem() (*Problem, error) {
	if t.Problem != nil {
		return t.Problem, nil
	}

	problem, err := t.db.Problem(t.ctx, t.ProblemID)
	if err != nil {
		return nil, err
	}

	t.Problem = problem
	return problem, nil
}

func (t *Test) SetScore(score int32) error {
	if err := t.db.raw.SetPbTestScore(t.ctx, rawdb.SetPbTestScoreParams{ProblemID: t.ProblemID, VisibleID: t.VisibleID, Score: score}); err != nil {
		return err
	}

	t.Score = score
	return nil
}

func (t *Test) SetVID(vid int64) error {
	if err := t.db.raw.SetPbTestVisibleID(t.ctx, rawdb.SetPbTestVisibleIDParams{ProblemID: t.ProblemID, OldID: t.VisibleID, NewID: vid}); err != nil {
		return err
	}

	t.VisibleID = vid
	return nil
}

// Test returns a test from the problem with id `pbID` and visible id `testVID`
func (db *DB) Test(ctx context.Context, pbID int64, testVID int64) (*Test, error) {
	test, err := db.raw.TestVisibleID(ctx, rawdb.TestVisibleIDParams{ProblemID: pbID, VisibleID: testVID})
	if err != nil {
		return nil, err
	}

	return db.testFromRaw(ctx, test), nil
}

func (db *DB) Tests(ctx context.Context, pbID int64) ([]*Test, error) {
	tests, err := db.raw.ProblemTests(ctx, pbID)
	if err != nil {
		return nil, err
	}

	var t []*Test
	for _, test := range tests {
		t = append(t, db.testFromRaw(ctx, test))
	}
	return t, nil
}

// TestByID returns a test with the specified internal ID
// This might not be what you are looking for, you should check out DB.Test
func (db *DB) TestByID(ctx context.Context, testID int64) (*Test, error) {
	test, err := db.raw.Test(ctx, testID)
	if err != nil {
		return nil, err
	}

	return db.testFromRaw(ctx, test), nil
}

func (db *DB) CreateTest(ctx context.Context, pid, vid int64, score int32) (*Test, error) {
	test, err := db.raw.CreateTest(ctx, rawdb.CreateTestParams{ProblemID: pid, VisibleID: vid, Score: score})
	if err != nil {
		return nil, err
	}
	return db.testFromRaw(ctx, test), nil
}

func (db *DB) testFromRaw(ctx context.Context, test rawdb.Test) *Test {
	return &Test{
		ctx: ctx,
		db:  db,

		ID:        test.ID,
		CreatedAt: test.CreatedAt,
		Score:     test.Score,
		ProblemID: test.ProblemID,
		VisibleID: test.VisibleID,
		Orphaned:  test.Orphaned,
	}
}
