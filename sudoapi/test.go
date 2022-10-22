package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) Test(ctx context.Context, pbID int, testVID int) (*kilonova.Test, *StatusError) {
	test, err := s.db.Test(ctx, pbID, testVID)
	if err != nil || test == nil {
		return nil, WrapError(ErrNotFound, "Test not found")
	}
	return test, nil
}

func (s *BaseAPI) TestByID(ctx context.Context, id int) (*kilonova.Test, *StatusError) {
	test, err := s.db.TestByID(ctx, id)
	if err != nil || test == nil {
		return nil, WrapError(ErrNotFound, "Test not found")
	}
	return test, nil
}

func (s *BaseAPI) Tests(ctx context.Context, pbID int) ([]*kilonova.Test, *StatusError) {
	tests, err := s.db.Tests(ctx, pbID)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return tests, nil
}

func (s *BaseAPI) UpdateTest(ctx context.Context, testID int, upd kilonova.TestUpdate) *StatusError {
	if err := s.db.UpdateTest(ctx, testID, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update test")
	}
	return nil
}

func (s *BaseAPI) CreateTest(ctx context.Context, test *kilonova.Test) *StatusError {
	if err := s.db.CreateTest(ctx, test); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create test")
	}
	return nil
}

func (s *BaseAPI) OrphanTests(ctx context.Context, problemID int) *StatusError {
	if err := s.db.OrphanProblemTests(ctx, problemID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't remove tests")
	}
	if err := s.CleanupSubTasks(ctx, problemID); err != nil {
		return err
	}
	return nil
}

// Please note that this function does not properly ensure that subtasks would be cleaned up afterwards.
// This is left as an exercise to the caller
func (s *BaseAPI) OrphanTest(ctx context.Context, id int) *StatusError {
	if err := s.db.OrphanTest(ctx, id); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't remove test")
	}
	return nil
}

func (s *BaseAPI) NextVID(ctx context.Context, problemID int) int {
	max, err := s.db.BiggestVID(ctx, problemID)
	if err != nil {
		max = 0
	}
	if max <= 0 {
		return 1
	}
	return max + 1
}
