package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
)

func (s *API) getPaste(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("pasteID")

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
