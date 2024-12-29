package test

import (
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

func ProcessSubmissionFile(ctx *ArchiveCtx, fpath string, r io.Reader, base *sudoapi.BaseAPI) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("couldn't read submission file: %w", err)
	}

	lang := base.LanguageFromFilename(ctx.ctx, path.Base(fpath))
	if lang == "" {
		if !strings.HasSuffix(fpath, ".desc") { // Don't show for polygon description files
			slog.WarnContext(ctx.ctx, "Unrecognized submisison language for archive file", slog.String("filename", path.Base(fpath)))
		}
		return nil
	}

	ctx.submissions = append(ctx.submissions, &submissionStub{
		code: data,
		lang: lang,
	})
	return nil
}
