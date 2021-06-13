package api

import (
	"io"
	"net/http"

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
	data, err := io.ReadAll(file)
	if err != nil {
		errorData(w, err, 500)
		return
	}
	name := fh.Filename
	if name == "" {
		name = "untitled.txt"
	}
	if args.Private {
		args.Visible = false
	}
	att := kilonova.Attachment{
		ProblemID: util.Problem(r).ID,
		Visible:   args.Visible,
		Private:   args.Private,
		Name:      name,
		Data:      data,
	}

	if err := s.db.CreateAttachment(r.Context(), &att); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, att.ID)
}

func (s *API) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	var removedAtts int
	atts, ok := DecodeIntString(r.FormValue("atts"))
	if !ok || len(atts) == 0 {
		errorData(w, "Invalid int string", 400)
		return
	}

	for _, attid := range atts {
		att, err := s.db.Attachment(r.Context(), attid)
		if err != nil {
			continue
		}
		if att.ProblemID != util.Problem(r).ID {
			continue
		}
		if err := s.db.DeleteAttachment(r.Context(), attid); err == nil {
			removedAtts++
		}
	}

	if removedAtts != len(atts) {
		errorData(w, "Some attachments could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected attachments")
}

func (s *API) getAttachments(w http.ResponseWriter, r *http.Request) {
	att, err := s.db.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, att)
}
