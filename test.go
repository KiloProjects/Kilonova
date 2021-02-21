package kilonova

import (
	"context"
	"time"
)

type Test struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Score     int       `json:"score"`
	ProblemID int       `db:"problem_id" json:"problem_id"`
	VisibleID int       `db:"visible_id" json:"visible_id"`
	Orphaned  bool      `json:"orphaned"`
}

type TestUpdate struct {
	Score     *int  `json:"score"`
	VisibleID *int  `json:"visible_id"`
	Orphaned  *bool `json:"orphaned"`
}

type TestService interface {
	CreateTest(ctx context.Context, test *Test) error

	Test(ctx context.Context, problemID, testVID int) (*Test, error)
	TestByID(ctx context.Context, id int) (*Test, error)
	Tests(ctx context.Context, problemID int) ([]*Test, error)

	UpdateTest(ctx context.Context, id int, upd TestUpdate) error

	// We don't delete tests, we orphan them
	// DeleteTest(ctx context.Context, id int) error

	OrphanProblemTests(ctx context.Context, problemID int) error
	BiggestVID(ctx context.Context, problemID int) (int, error)
}
