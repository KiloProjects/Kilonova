package api

import (
	"io"
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
		Exec    bool `json:"exec"`
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
		Exec:    args.Exec,
		Name:    name,
	}

	if err := s.base.CreateAttachment(r.Context(), &att, util.Problem(r).ID, file); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, att.ID)
}

func (s *API) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	var atts []int
	if err := parseJsonBody(r, &atts); err != nil {
		err.WriteError(w)
		return
	}

	removedAtts, err1 := s.base.DeleteAttachments(r.Context(), util.Problem(r).ID, atts)
	if err1 != nil {
		errorData(w, "Error deleting attachments", 500)
		return
	}

	if removedAtts != len(atts) {
		errorData(w, "Some attachments could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected attachments")
}

func (s *API) updateAttachmentData(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Name string `json:"name"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	if args.Name == "" {
		errorData(w, "You must provide a name", 400)
		return
	}

	att, err1 := s.base.AttachmentByName(r.Context(), util.Problem(r).ID, args.Name)
	if err1 != nil {
		err1.WriteError(w)
		return
	}

	file, _, err := r.FormFile("data")
	if err != nil {
		errorData(w, err, 400)
		return
	}
	defer file.Close()

	val, err := io.ReadAll(file)
	if err != nil {
		errorData(w, err, 400)
		return
	}

	if err := s.base.UpdateAttachmentData(r.Context(), att.ID, val); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Updated attachment data")
}

func (s *API) bulkUpdateAttachmentInfo(w http.ResponseWriter, r *http.Request) {
	var data map[int]struct {
		Name    *string `json:"name"`
		Visible *bool   `json:"visible"`
		Private *bool   `json:"private"`
		Exec    *bool   `json:"exec"`
	}
	var updatedAttachments int

	if err := parseJsonBody(r, &data); err != nil {
		err.WriteError(w)
		return
	}

	// Ensure only the selected problem's attachments are updated
	atts, err := s.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}
	for _, att := range atts {
		if val, ok := data[att.ID]; ok {
			if err := s.base.UpdateAttachment(r.Context(), att.ID, &kilonova.AttachmentUpdate{
				Visible: val.Visible,
				Private: val.Private,
				Exec:    val.Exec,
				Name:    val.Name,
			}); err == nil {
				updatedAttachments++
			}
		}
	}

	if updatedAttachments != len(data) {
		errorData(w, "Some attachments could not be updated", 500)
		return
	}
	returnData(w, "Updated all attachment metadata")
}
