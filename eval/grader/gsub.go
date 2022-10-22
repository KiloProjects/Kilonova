package grader

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
)

var _ eval.GraderSubmission = &gsub{}

type gsub struct {
	sub *kilonova.Submission
	h   *Handler
}

func (g *gsub) Submission() *kilonova.Submission {
	return g.sub
}

func (g *gsub) Update(upd kilonova.SubmissionUpdate) error {
	if err := g.h.base.UpdateSubmission(context.Background(), g.sub.ID, upd); err != nil {
		return err
	}
	return nil
}

func (g *gsub) Close() error { return nil }

func (h *Handler) makeGraderSub(sub *kilonova.Submission) eval.GraderSubmission {
	return &gsub{sub, h}
}
