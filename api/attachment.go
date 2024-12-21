package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createAttachment(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(50 * 1024 * 1024) // 50MB
	defer cleanupMultipart(r)
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

	if util.Problem(r) != nil {
		if err := s.base.CreateProblemAttachment(r.Context(), &att, util.Problem(r).ID, file, &util.UserBrief(r).ID); err != nil {
			statusError(w, err)
			return
		}
		returnData(w, att.ID)
	} else if util.BlogPost(r) != nil {
		if err := s.base.CreateBlogPostAttachment(r.Context(), &att, util.BlogPost(r).ID, file, &util.UserBrief(r).ID); err != nil {
			statusError(w, err)
			return
		}
		returnData(w, att.ID)
	} else {
		slog.ErrorContext(r.Context(), "Invalid attachment context")
	}
}

func (s *API) bulkDeleteAttachments(w http.ResponseWriter, r *http.Request) {
	var atts []int
	if err := parseJSONBody(r, &atts); err != nil {
		statusError(w, err)
		return
	}

	var removedAtts int
	var err error
	if util.Problem(r) != nil {
		removedAtts, err = s.base.DeleteProblemAtts(r.Context(), util.Problem(r).ID, atts)
	} else if util.BlogPost(r) != nil {
		removedAtts, err = s.base.DeleteBlogPostAtts(r.Context(), util.BlogPost(r).ID, atts)
	} else {
		slog.ErrorContext(r.Context(), "Invalid attachment context")
		return
	}

	if err != nil {
		slog.ErrorContext(r.Context(), "Couldn't delete attachments", slog.Any("err", err))
		errorData(w, "Error deleting attachments", 500)
		return
	}

	if removedAtts != len(atts) {
		errorData(w, "Some attachments could not be deleted", 500)
		return
	}
	returnData(w, "Deleted selected attachments")
}

func cleanupMultipart(r *http.Request) {
	if r.MultipartForm == nil {
		return
	}
	if err := r.MultipartForm.RemoveAll(); err != nil {
		slog.ErrorContext(r.Context(), "Could not clean up multipart form", slog.Any("err", err))
	}
}

func (s *API) updateAttachmentData(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(20 * 1024 * 1024)
	defer cleanupMultipart(r)
	var args struct {
		ID int `json:"id"`

		Name    *string `json:"name"`
		Visible *bool   `json:"visible"`
		Private *bool   `json:"private"`
		Exec    *bool   `json:"exec"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}
	if args.ID <= 0 {
		errorData(w, "You must provide an id", 400)
		return
	}

	var att *kilonova.Attachment
	if util.Problem(r) != nil {
		att1, err := s.base.ProblemAttachment(r.Context(), util.Problem(r).ID, args.ID)
		if err != nil {
			statusError(w, err)
			return
		}
		att = att1
	} else if util.BlogPost(r) != nil {
		att1, err := s.base.BlogPostAttachment(r.Context(), util.BlogPost(r).ID, args.ID)
		if err != nil {
			statusError(w, err)
			return
		}
		att = att1
	} else {
		slog.ErrorContext(r.Context(), "Invalid attachment context")
		return
	}

	if err := s.base.UpdateAttachment(r.Context(), att.ID, &kilonova.AttachmentUpdate{
		Visible: args.Visible,
		Private: args.Private,
		Exec:    args.Exec,
		Name:    args.Name,
	}); err != nil && !errors.Is(err, kilonova.ErrNoUpdates) {
		statusError(w, err)
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

	if err := s.base.UpdateAttachmentData(r.Context(), att.ID, val, util.UserBrief(r)); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Updated attachment")
}

type fullAttachment struct {
	Metadata *kilonova.Attachment `json:"metadata"`
	MimeType string               `json:"mime_type"`
	Data     []byte               `json:"data"`
}

// NOTE: This depends on the middleware. The middleware actually resolves the attachment, either by name or by id.
func (s *API) getFullAttachment(ctx context.Context, _ struct{}) (*fullAttachment, error) {
	data, err := s.base.AttachmentData(ctx, util.AttachmentContext(ctx).ID)
	if err != nil {
		return nil, err
	}
	return &fullAttachment{
		Metadata: util.AttachmentContext(ctx),
		MimeType: http.DetectContentType(data),
		Data:     data,
	}, nil
}

func (s *API) bulkUpdateAttachmentInfo(w http.ResponseWriter, r *http.Request) {
	var data map[int]struct {
		Name    *string `json:"name"`
		Visible *bool   `json:"visible"`
		Private *bool   `json:"private"`
		Exec    *bool   `json:"exec"`
	}
	var updatedAttachments int

	if err := parseJSONBody(r, &data); err != nil {
		statusError(w, err)
		return
	}

	// Ensure only the selected problem/blogpost attachments are updated
	var atts []*kilonova.Attachment
	if util.Problem(r) != nil {
		atts1, err := s.base.ProblemAttachments(r.Context(), util.Problem(r).ID)
		if err != nil {
			statusError(w, err)
			return
		}
		atts = atts1
	} else if util.BlogPost(r) != nil {
		atts1, err := s.base.BlogPostAttachments(r.Context(), util.BlogPost(r).ID)
		if err != nil {
			statusError(w, err)
			return
		}
		atts = atts1
	} else {
		slog.ErrorContext(r.Context(), "Invalid attachment context")
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
