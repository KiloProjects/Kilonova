package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) Test(ctx context.Context, pbID int, testVID int) (*kilonova.Test, error) {
	test, err := s.db.Test(ctx, pbID, testVID)
	if err != nil || test == nil {
		return nil, fmt.Errorf("test not found: %w", ErrNotFound)
	}
	return test, nil
}

func (s *BaseAPI) Tests(ctx context.Context, pbID int) ([]*kilonova.Test, error) {
	tests, err := s.db.Tests(ctx, pbID)
	if err != nil {
		slog.WarnContext(ctx, "couldn't get tests", slog.Int("problemID", pbID), slog.Any("err", err))
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		return nil, fmt.Errorf("couldn't get test: %w", err)
	}
	return tests, nil
}

func (s *BaseAPI) UpdateTest(ctx context.Context, testID int, upd kilonova.TestUpdate) error {
	if err := s.db.UpdateTest(ctx, testID, upd); err != nil {
		slog.WarnContext(ctx, "couldn't update test", slog.Int("testID", testID), slog.Any("err", err))
		return fmt.Errorf("couldn't update test: %w", err)
	}
	return nil
}

func (s *BaseAPI) CreateTest(ctx context.Context, test *kilonova.Test) error {
	if err := s.db.CreateTest(ctx, test); err != nil {
		slog.WarnContext(ctx, "couldn't create test", slog.Any("testID", test), slog.Any("err", err))
		return fmt.Errorf("couldn't create test: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteTests(ctx context.Context, problemID int) error {
	ids, err := s.db.DeleteProblemTests(ctx, problemID)
	if err != nil {
		slog.WarnContext(ctx, "couldn't remove tests", slog.Int("problemID", problemID), slog.Any("err", err))
		return fmt.Errorf("couldn't remove tests: %w", err)
	}
	for _, id := range ids {
		if err := s.PurgeTestData(id); err != nil {
			slog.WarnContext(ctx, "couldn't remove test data", slog.Int("testID", id), slog.Any("err", err))
		}
	}
	if err := s.CleanupSubTasks(ctx, problemID); err != nil {
		return err
	}
	return nil
}

// Please note that this function does not properly ensure that subtasks would be cleaned up afterwards.
// This is left as an exercise to the caller
func (s *BaseAPI) DeleteTest(ctx context.Context, id int) error {
	if err := s.db.DeleteTest(ctx, id); err != nil {
		slog.WarnContext(ctx, "couldn't remove test", slog.Int("testID", id), slog.Any("err", err))
		return fmt.Errorf("couldn't remove test: %w", err)
	}
	if err := s.PurgeTestData(id); err != nil {
		slog.WarnContext(ctx, "couldn't remove test data", slog.Int("testID", id), slog.Any("err", err))
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
