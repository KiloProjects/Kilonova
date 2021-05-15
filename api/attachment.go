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
		Visible bool   `json:"visible"`
		Name    string `json:"name,required"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	file, _, err := r.FormFile("data")
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
	att := kilonova.Attachment{
		ProblemID: util.Problem(r).ID,
		Visible:   args.Visible,
		Name:      args.Name,
		Data:      data,
	}

	if err := s.aserv.CreateAttachment(r.Context(), &att); err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, att.ID)
}

func (s *API) updateAttachmentMetadata(w http.ResponseWriter, r *http.Request) {
	//TODO
}

/*
func (s *API) saveAttachmentData(w http.ResponseWriter, r *http.Request) {
	//TODO
}
*/

func (s *API) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func (s *API) getAttachment(w http.ResponseWriter, r *http.Request) {
	//TODO
}

func (s *API) getAttachments(w http.ResponseWriter, r *http.Request) {
	att, err := s.aserv.Attachments(r.Context(), false, kilonova.AttachmentFilter{ProblemID: &util.Problem(r).ID})
	if err != nil {
		errorData(w, err, 500)
		return
	}
	returnData(w, att)
}
