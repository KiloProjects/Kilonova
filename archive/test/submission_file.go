package test

import (
	"archive/zip"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
)

type submissionStub struct {
	code []byte
	lang string
}

func ProcessSubmissionFile(ctx *ArchiveCtx, file *zip.File, base *sudoapi.BaseAPI) error {
	f, err := file.Open()
	if err != nil {
		return kilonova.WrapError(err, "Couldn't open submission file")
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't read submission file")
	}

	lang := base.LanguageFromFilename(ctx.ctx, path.Base(file.Name))
	if lang == "" {
		if !strings.HasSuffix(file.Name, ".desc") { // Don't show for polygon description files
			slog.WarnContext(ctx.ctx, "Unrecognized submisison language for archive file", slog.String("filename", path.Base(file.Name)))
		}
		return nil
	}

	ctx.submissions = append(ctx.submissions, &submissionStub{
		code: data,
		lang: lang,
	})
	return nil
}
