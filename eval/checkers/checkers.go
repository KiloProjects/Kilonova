package checkers

import (
	"context"
	"io"

	"github.com/shopspring/decimal"
)

// Checker is an interface for a function that statelessly tries to evaluate a subtest from a submission
type Checker interface {
	Prepare(context.Context) (string, error)
	Cleanup(context.Context) error

	// RunChecker returns a comment and a decimal number [0, 100] signifying the percentage of correctness of the subtest
	RunChecker(ctx context.Context, programOut, correctInput, correctOut io.Reader) (string, decimal.Decimal)
}
