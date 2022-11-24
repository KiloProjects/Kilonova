package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/go-chi/chi/v5"
)

func (s *API) createPaste(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		SubmissionID int `json:"id"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	sub, err := s.base.RawSubmission(r.Context(), args.SubmissionID)
	if err != nil {
		err.WriteError(w)
		return
	}
	if !util.IsSubmissionEditor(sub, util.UserBrief(r)) {
		errorData(w, "You can't create a paste for this submission!", 403)
		return
	}

	id, err := s.base.CreatePaste(r.Context(), sub, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, id)
}

func (s *API) getPaste(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "pasteID")

	paste, err := s.base.SubmissionPaste(r.Context(), id)
	if err != nil {
		err.WriteError(w)
		return
	}

	sub, err := s.fullSubmission(r.Context(), paste.Submission.ID, nil, false)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		ID         string              `json:"id"`
		Submission *subLine            `json:"sub"`
		Author     *kilonova.UserBrief `json:"author"`
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
		err.WriteError(w)
		return
	}

	if !util.IsPasteEditor(paste, util.UserBrief(r)) {
		errorData(w, "You can't delete this paste", 403)
		return
	}

	if err := s.base.DeletePaste(r.Context(), paste.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Deleted.")
}
