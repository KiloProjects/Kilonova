package sudoapi

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova"
)

func (s *BaseAPI) SubTasks(ctx context.Context, problemID int) ([]*kilonova.SubTask, error) {
	stks, err := s.db.SubTasks(ctx, problemID)
	if err != nil {
		slog.WarnContext(ctx, "couldn't get subtasks", slog.Int("problemID", problemID), slog.Any("err", err))
		return nil, ErrUnknownError
	}
	return stks, nil
}

func (s *BaseAPI) SubTasksByTest(ctx context.Context, problemID, testID int) ([]*kilonova.SubTask, error) {
	stks, err := s.db.SubTasksByTest(ctx, problemID, testID)
	if err != nil {
		slog.WarnContext(ctx, "couldn't get subtasks", slog.Int("problemID", problemID), slog.Int("testID", testID), slog.Any("err", err))
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
		slog.WarnContext(ctx, "couldn't create subtask", slog.Any("subtask", subtask), slog.Any("err", err))
		return fmt.Errorf("couldn't create subtask: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateSubTask(ctx context.Context, id int, upd kilonova.SubTaskUpdate) error {
	if err := s.db.UpdateSubTask(ctx, id, upd); err != nil {
		slog.WarnContext(ctx, "couldn't update subtask", slog.Int("subtaskID", id), slog.Any("err", err))
		return fmt.Errorf("couldn't update subtask metadata: %w", err)
	}
	return nil
}

func (s *BaseAPI) UpdateSubTaskTests(ctx context.Context, id int, testIDs []int) error {
	if err := s.db.UpdateSubTaskTests(ctx, id, testIDs); err != nil {
		slog.WarnContext(ctx, "couldn't update subtask tests", slog.Int("subtaskID", id), slog.Any("err", err))
		return fmt.Errorf("couldn't update subtask tests: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteSubTask(ctx context.Context, subtaskID int) error {
	if err := s.db.DeleteSubTask(ctx, subtaskID); err != nil {
		slog.WarnContext(ctx, "couldn't delete subtask", slog.Int("subtaskID", subtaskID), slog.Any("err", err))
		return fmt.Errorf("couldn't delete subtask: %w", err)
	}
	return nil
}

func (s *BaseAPI) DeleteSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.DeleteSubTasks(ctx, problemID); err != nil {
		slog.WarnContext(ctx, "couldn't delete subtasks", slog.Int("problemID", problemID), slog.Any("err", err))
		return fmt.Errorf("couldn't remove subtasks: %w", err)
	}
	return nil
}

// CleanupSubtasks removes all subtasks from a problem that do not have any tests
func (s *BaseAPI) CleanupSubTasks(ctx context.Context, problemID int) error {
	if err := s.db.CleanupSubTasks(ctx, problemID); err != nil {
		slog.WarnContext(ctx, "couldn't clean up subtasks", slog.Int("problemID", problemID), slog.Any("err", err))
		return fmt.Errorf("couldn't clean up subtasks: %w", err)
	}
	return nil
}
