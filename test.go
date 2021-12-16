package kilonova

import (
	"time"

	"github.com/jackc/pgtype"
)

type Test struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	Score     int       `json:"score"`
	ProblemID int       `db:"problem_id" json:"problem_id"`
	VisibleID int       `db:"visible_id" json:"visible_id"`
	Orphaned  bool      `json:"orphaned"`

	InputOID  pgtype.OID `json:"-" db:"input_oid"`
	OutputOID pgtype.OID `json:"-" db:"output_oid"`
}

type TestUpdate struct {
	Score     *int  `json:"score"`
	VisibleID *int  `json:"visible_id"`
	Orphaned  *bool `json:"orphaned"`
}

type SubTask struct {
	ID        int       `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	ProblemID int       `json:"problem_id"`
	VisibleID int       `json:"visible_id"`
	Score     int       `json:"score"`
	Tests     []int     `json:"tests"`
}

type SubTaskUpdate struct {
	VisibleID *int  `json:"visible_id"`
	Score     *int  `json:"score"`
	Tests     []int `json:"tests"`
}
