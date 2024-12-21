package sudoapi

import (
	"context"

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
		return nil, WrapError(ErrNotFound, "Couldn't find subtask")
	}
	return stk, nil
}

func (s *BaseAPI) CreateSubTask(ctx context.Context, subtask *kilonova.SubTask) error {
	if err := s.db.CreateSubTask(ctx, subtask); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't create subtask")
	}
	return nil
}

func (s *BaseAPI) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
	if err := s.db.UpdateSubTask(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update subtask metadata")
	}
	return nil
}

func (s *BaseAPI) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	if err := s.db.UpdateSubTaskTests(ctx, id, testIDs); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update subtask tests")
	}
	return nil
}

func (s *BaseAPI) DeleteSubTask(ctx context.Context, subtaskID int) error {
	if err := s.db.DeleteSubTask(ctx, subtaskID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't delete subtask")
	}
	return nil
}

func (s *BaseAPI) DeleteSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.DeleteSubTasks(ctx, problemID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't remove subtasks")
	}
	return nil
}

// CleanupSubtasks removes all subtasks from a problem that do not have any tests
func (s *BaseAPI) CleanupSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.CleanupSubTasks(ctx, problemID); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't clean up subtasks")
	}
	return nil
}
