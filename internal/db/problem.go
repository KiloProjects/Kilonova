package db

import (
	"context"
	"time"

	"github.com/KiloProjects/Kilonova/internal/rawdb"
	"github.com/gosimple/slug"
)

// Problem provides information and utility functions regarding problems
// Note that it should only be used only while the context you got the problem from is not cancelled
type Problem struct {
	ctx   context.Context
	db    *DB
	Empty bool `json:"-"`

	ID           int64     `json:"id"`
	CreatedAt    time.Time `json:"created_at"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	TestName     string    `json:"test_name"`
	Author       *User     `json:"author,omitempty"`
	AuthorID     int64     `json:"author_id"`
	TimeLimit    float64   `json:"time_limit"`
	MemoryLimit  int32     `json:"memory_limit"`
	StackLimit   int32     `json:"stack_limit"`
	SourceSize   int32     `json:"source_size"`
	ConsoleInput bool      `json:"console_input"`
	Visible      bool      `json:"visible"`
}

func (p *Problem) SetName(name string) error {
	if err := p.db.raw.SetProblemName(p.ctx, rawdb.SetProblemNameParams{ID: p.ID, Name: name}); err != nil {
		return err
	}

	p.Name = name
	return nil
}

func (p *Problem) SetDescription(desc string) error {
	if err := p.db.raw.SetProblemDescription(p.ctx, rawdb.SetProblemDescriptionParams{ID: p.ID, Description: desc}); err != nil {
		return err
	}

	p.Description = desc
	return nil
}

func (p *Problem) SetVisibility(visible bool) error {
	if err := p.db.raw.SetProblemVisibility(p.ctx, rawdb.SetProblemVisibilityParams{ID: p.ID, Visible: visible}); err != nil {
		return err
	}

	p.Visible = visible
	return nil
}

func (p *Problem) SetTestName(testName string) error {
	if err := p.db.raw.SetTestName(p.ctx, rawdb.SetTestNameParams{ID: p.ID, TestName: testName}); err != nil {
		return err
	}

	p.TestName = testName
	return nil
}

func (p *Problem) SetConsoleInput(cInput bool) error {
	if err := p.db.raw.SetConsoleInput(p.ctx, rawdb.SetConsoleInputParams{ID: p.ID, ConsoleInput: cInput}); err != nil {
		return err
	}

	p.ConsoleInput = cInput
	return nil
}

func (p *Problem) SetLimits(memoryLimit, stackLimit int32, timeLimit float64) error {
	if err := p.db.raw.SetLimits(p.ctx, rawdb.SetLimitsParams{ID: p.ID, MemoryLimit: memoryLimit, StackLimit: stackLimit, TimeLimit: timeLimit}); err != nil {
		return err
	}

	p.MemoryLimit = memoryLimit
	p.StackLimit = stackLimit
	p.TimeLimit = timeLimit
	return nil
}

func (p *Problem) BiggestVID() (int64, error) {
	return p.db.raw.BiggestVID(p.ctx, p.ID)
}

func (p *Problem) ClearTests() error {
	return p.db.raw.PurgePbTests(p.ctx, p.ID)
}

func (p *Problem) MaxScore(uid int64) int {
	return p.db.MaxScore(p.ctx, uid, p.ID)
}

// GetAuthor returns the author of the specified problem
// Additionally, it fills Problem.Author
func (p *Problem) GetAuthor() (*User, error) {
	if p.AuthorID <= 0 {
		return nil, nil
	}
	author, err := p.db.User(p.ctx, p.AuthorID)
	if err != nil {
		return nil, err
	}
	p.Author = author
	return author, nil
}

// Test returns a test from the problem with its Problem field already filled
func (p *Problem) Test(vid int64) (*Test, error) {
	test, err := p.db.Test(p.ctx, p.ID, vid)
	if err != nil {
		return nil, err
	}

	test.Problem = p
	return test, nil
}

// Tests returns all tests from the problem with their Problem field already filled
func (p *Problem) Tests() ([]*Test, error) {
	tests, err := p.db.Tests(p.ctx, p.ID)
	if err != nil {
		return nil, err
	}

	for i := 0; i < len(tests); i++ {
		tests[i].Problem = p
	}
	return tests, nil
}

// Problem returns a problem with the specified id
func (db *DB) Problem(ctx context.Context, id int64) (*Problem, error) {
	pb, err := db.raw.Problem(ctx, id)
	if err != nil {
		return nil, err
	}

	return db.pbFromRaw(ctx, pb), nil
}

// ProblemByName returns a problem with the specified name (note that it is case insensitive)
func (db *DB) ProblemByName(ctx context.Context, name string) (*Problem, error) {
	pb, err := db.raw.ProblemByName(ctx, name)
	if err != nil {
		return nil, err
	}

	return db.pbFromRaw(ctx, pb), nil
}

// Problems returns all problems
func (db *DB) Problems(ctx context.Context) ([]*Problem, error) {
	pbs, err := db.raw.Problems(ctx)
	if err != nil {
		return nil, err
	}

	var problems []*Problem
	for _, problem := range pbs {
		problems = append(problems, db.pbFromRaw(ctx, problem))
	}

	return problems, nil
}

func (db *DB) CreateProblem(ctx context.Context, name string, authorID int64, consoleInput bool) (*Problem, error) {
	pb, err := db.raw.CreateProblem(ctx, rawdb.CreateProblemParams{
		Name:         name,
		AuthorID:     authorID,
		ConsoleInput: consoleInput,
		TestName:     slug.Make(name),
		MemoryLimit:  65536, // 64 * 1024KB = 64MB
		StackLimit:   16384, // 16 * 1024KB = 16MB
		TimeLimit:    0.1,   // 0.1s
	})
	if err != nil {
		return nil, err
	}

	return db.pbFromRaw(ctx, pb), nil
}

func (db *DB) VisibleProblems(ctx context.Context, user *User) ([]*Problem, error) {
	if user.Admin {
		return db.Problems(ctx)
	}

	pbs, err := db.raw.VisibleProblems(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	var problems []*Problem
	for _, problem := range pbs {
		problems = append(problems, db.pbFromRaw(ctx, problem))
	}

	return problems, nil
}

func (db *DB) pbFromRaw(ctx context.Context, pb rawdb.Problem) *Problem {
	return &Problem{
		ctx: ctx,
		db:  db,

		ID:           pb.ID,
		CreatedAt:    pb.CreatedAt,
		Name:         pb.Name,
		Description:  pb.Description,
		TestName:     pb.TestName,
		AuthorID:     pb.AuthorID,
		TimeLimit:    pb.TimeLimit,
		MemoryLimit:  pb.MemoryLimit,
		StackLimit:   pb.StackLimit,
		SourceSize:   pb.SourceSize,
		ConsoleInput: pb.ConsoleInput,
		Visible:      pb.Visible,
	}
}
