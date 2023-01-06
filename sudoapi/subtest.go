package sudoapi

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

func (s *BaseAPI) SubTest(ctx context.Context, id int) (*kilonova.SubTest, *StatusError) {
	stest, err := s.db.SubTest(ctx, id)
	if err != nil || stest == nil {
		return nil, WrapError(ErrNotFound, "Couldn't find subtest")
	}
	return stest, nil
}

func (s *BaseAPI) SubTests(ctx context.Context, submissionID int) ([]*kilonova.SubTest, *StatusError) {
	stests, err := s.db.SubTestsBySubID(ctx, submissionID)
	if err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		return nil, ErrUnknownError
	}
	return stests, nil
}

func (s *BaseAPI) UpdateSubTest(ctx context.Context, id int, upd kilonova.SubTestUpdate) *StatusError {
	if err := s.db.UpdateSubTest(ctx, id, upd); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't update subtest")
	}
	return nil
}
