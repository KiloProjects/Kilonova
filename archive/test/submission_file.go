package test

import (
	"archive/zip"
	"fmt"
	"io"
	"log/slog"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova/sudoapi"
)

type submissionStub struct {
	code []byte
	lang string
}

func ProcessSubmissionFile(ctx *ArchiveCtx, file *zip.File, base *sudoapi.BaseAPI) error {
	f, err := file.Open()
	if err != nil {
		return fmt.Errorf("Couldn't open submission file: %w", err)
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("Couldn't read submission file: %w", err)
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
