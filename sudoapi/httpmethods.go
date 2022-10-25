package sudoapi

import (
	"context"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/util"
)

// Methods that can be reused in the api/ package. Note that session stuff is in session.go

func (s *WebHandler) CreateSubmission(ctx context.Context, args struct {
	Code string `json:"code"`
	Lang string `json:"language"`
}) (int, *StatusError) {
	lang, ok := eval.Langs[args.Lang]
	if !ok {
		return -1, Statusf(400, "Invalid language")
	}
	return s.base.CreateSubmission(ctx, util.UserBriefContext(ctx), util.ProblemContext(ctx), args.Code, lang)
}

func (s *WebHandler) DeleteSubmission(ctx context.Context, args struct {
	SubmissionID int `json:"submission_id"`
}) (string, *StatusError) {
	if err := s.base.DeleteSubmission(ctx, args.SubmissionID); err != nil {
		return "", err
	}
	return "Deleted submission", nil
}
