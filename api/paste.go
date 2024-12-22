package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
)

func (s *API) createPaste(w http.ResponseWriter, r *http.Request) {
	if !util.Submission(r).IsEditor(util.UserBrief(r)) {
		errorData(w, "You can't create a paste for this submission!", 403)
		return
	}

	id, err := s.base.CreatePaste(r.Context(), &util.Submission(r).Submission, util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, id)
}

func (s *API) getPaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pasteID")

	paste, err := s.base.SubmissionPaste(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}

	sub, err := s.fullSubmission(r.Context(), paste.Submission.ID, nil, false)
	if err != nil {
		statusError(w, err)
		return
	}

	returnData(w, struct {
		ID         string                  `json:"id"`
		Submission *sudoapi.FullSubmission `json:"sub"`
		Author     *kilonova.UserBrief     `json:"author"`
	}{
		ID:         paste.ID,
		Submission: sub,
		Author:     paste.Author,
	})
}

func (s *API) deletePaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pasteID")

	paste, err := s.base.SubmissionPaste(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}

	if !paste.IsEditor(util.UserBrief(r)) {
		errorData(w, "You can't delete this paste", 403)
		return
	}

	if err := s.base.DeletePaste(r.Context(), paste.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Deleted.")
}
