package test

import (
	"archive/zip"
	"encoding/json"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

type archiveAttachment struct {
	File    *zip.File
	Name    string
	Visible bool
	Private bool
	Exec    bool
}

type attachmentProps struct {
	Visible bool `json:"visible"`
	Private bool `json:"private"`
	Exec    bool `json:"exec"`
}

func ProcessAttachmentFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	name := path.Base(file.Name)
	if strings.HasSuffix(name, ".att_props") {
		// Parse attachment props
		// TODO: Bring property autocomplete from frontend for attachments that don't have .att_props

		var props attachmentProps

		name = strings.TrimSuffix(name, ".att_props")

		f, err := file.Open()
		if err != nil {
			return kilonova.WrapError(err, "Couldn't open props file")
		}
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&props); err != nil {
			return kilonova.WrapError(err, "Invalid props file")
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
		val.File = file
		ctx.attachments[name] = val
	} else {
		ctx.attachments[name] = archiveAttachment{
			File:    file,
			Name:    name,
			Visible: false,
			Private: false,
			Exec:    false,
		}
	}
	return nil
}

func ProcessPolygonCheckFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	ctx.attachments["checker.cpp17"] = archiveAttachment{
		File:    file,
		Name:    "checker.cpp17",
		Visible: false,
		Private: true,
		Exec:    true,
	}
	return nil
}

func ProcessPolygonPDFStatement(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	parts := strings.Split(file.Name, "/")
	if len(parts) != 4 {
		zap.S().Warn("Sanity check failed: Polygon PDF statement is not 4 parts")
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
		File:    file,
		Name:    filename,
		Visible: false,
		Private: false,
		Exec:    false,
	}
	return nil
}
