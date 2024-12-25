package sudoapi

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) SubTasks(ctx context.Context, problemID int) ([]*kilonova.SubTask, error) {
	stks, err := s.db.SubTasks(ctx, problemID)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return stks, nil
}

func (s *BaseAPI) SubTasksByTest(ctx context.Context, problemID, testID int) ([]*kilonova.SubTask, error) {
	stks, err := s.db.SubTasksByTest(ctx, problemID, testID)
	if err != nil {
		zap.S().Warn(err)
		return nil, ErrUnknownError
	}
	return stks, nil
}

func (s *BaseAPI) SubTask(ctx context.Context, problemID int, subtaskVID int) (*kilonova.SubTask, error) {
	stk, err := s.db.SubTask(ctx, problemID, subtaskVID)
	if err != nil || stk == nil {
		return nil, fmt.Errorf("couldn't find subtask: %w", ErrNotFound)
	}
	return stk, nil
}

func (s *BaseAPI) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if err := s.db.CreateSubTask(ctx, subtask); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't create subtask: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
	if err := s.db.UpdateSubTask(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't update subtask metadata: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	if err := s.db.UpdateSubTaskTests(ctx, id, testIDs); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't update subtask tests: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteSubTask(ctx context.Context, subtaskID int) error {
	if err := s.db.DeleteSubTask(ctx, subtaskID); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't delete subtask: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.DeleteSubTasks(ctx, problemID); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't remove subtasks: %w", err)
	}
	return nil
}

// CleanupSubtasks removes all subtasks from a problem that do not have any tests
func (s *BaseAPI) CleanupSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.CleanupSubTasks(ctx, problemID); err != nil {
		zap.S().Warn(err)
		return fmt.Errorf("couldn't clean up subtasks: %w", err)
	}
	return nil
}
