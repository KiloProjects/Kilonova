package test

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strings"
)

type archiveAttachment struct {
	FilePath string
	Name     string
	Visible  bool
	Private  bool
	Exec     bool
}

type attachmentProps struct {
	Visible bool `json:"visible"`
	Private bool `json:"private"`
	Exec    bool `json:"exec"`
}

func ProcessAttachmentFile(ctx *ArchiveCtx, fpath string) error {
	name := path.Base(fpath)
	if strings.HasSuffix(name, ".att_props") {
		// Parse attachment props
		// TODO: Bring property autocomplete from frontend for attachments that don't have .att_props

		var props attachmentProps

		name = strings.TrimSuffix(name, ".att_props")

		data, err := fs.ReadFile(ctx.fs, name)
		if err != nil {
			return fmt.Errorf("couldn't read props file: %w", err)
		}
		if err := json.Unmarshal(data, &props); err != nil {
			return fmt.Errorf("invalid props file: %w", err)
		}

		_, ok := ctx.attachments[name]
		if ok {
			val := ctx.attachments[name]
			val.Visible = props.Visible
			val.Private = props.Private
			val.Exec = props.Exec
			ctx.attachments[name] = val
		} else {
			ctx.attachments[name] = archiveAttachment{
				Name:    name,
				Visible: props.Visible,
				Private: props.Private,
				Exec:    props.Exec,
			}
		}
		return nil
	}
	_, ok := ctx.attachments[name]
	if ok {
		val := ctx.attachments[name]
		val.FilePath = fpath
		ctx.attachments[name] = val
	} else {
		ctx.attachments[name] = archiveAttachment{
			FilePath: fpath,
			Name:     name,
			Visible:  false,
			Private:  false,
			Exec:     false,
		}
	}
	return nil
}

func ProcessPolygonCheckFile(ctx *ArchiveCtx, fpath string) error {
	ctx.attachments["checker.cpp17"] = archiveAttachment{
		FilePath: fpath,
		Name:     "checker.cpp17",
		Visible:  false,
		Private:  true,
		Exec:     true,
	}
	return nil
}

func ProcessPolygonPDFStatement(ctx *ArchiveCtx, fpath string) error {
	parts := strings.Split(fpath, "/")
	if len(parts) != 4 {
		slog.WarnContext(ctx.ctx, "Sanity check failed: Polygon PDF statement is not 4 parts")
		return nil
	}
	filename := ""
	switch parts[2] {
	case "english":
		filename = "statement-en.pdf"
	case "romanian":
		filename = "statement-ro.pdf"
	default:
		return nil
	}
	ctx.attachments[filename] = archiveAttachment{
		FilePath: fpath,
		Name:     filename,
		Visible:  false,
		Private:  false,
		Exec:     false,
	}
	return nil
}
