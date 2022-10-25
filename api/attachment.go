package api

import (
	"net/http"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createAttachment(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(50 * 1024 * 1024) // 50MB
	var args struct {
		Visible bool `json:"visible"`
		Private bool `json:"private"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	file, fh, err := r.FormFile("data")
	if err != nil {
		errorData(w, err, 400)
		return
	}
	defer file.Close()

	name := path.Base(path.Clean(fh.Filename))
	if name == "" || name == "/" || name == "." {
		name = "untitled.txt"
	}

	att := kilonova.Attachment{
		Visible: args.Visible,
		Private: args.Private,
		Name:    name,
	}

	if err := s.base.CreateAttachment(r.Context(), &att, util.Problem(r).ID, file); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, att.ID)
}

// TODO: Test
func (s *API) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	var atts []int
	if err := parseJsonBody(r, &atts); err != nil {
		err.WriteError(w)
		return
	}

	removedAtts, err := s.base.DeleteAttachments(r.Context(), util.Problem(r).ID, atts)
	if err != nil {
		errorData(w, "Error deleting attachments", 500)
		return
	}

	if removedAtts != len(atts) {
		errorData(w, "Some attachments could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected attachments")
}
