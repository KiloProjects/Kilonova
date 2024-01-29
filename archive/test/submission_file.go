package test

import (
	"archive/zip"
	"io"
	"path"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
)

type submissionStub struct {
	code []byte
	lang string
}

func ProcessSubmissionFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	f, err := file.Open()
	if err != nil {
		return kilonova.WrapError(err, "Couldn't open submission file")
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return kilonova.WrapError(err, "Couldn't read submission file")
	}

	lang := eval.GetLangByFilename(path.Base(file.Name))
	if lang == "" {
		if !strings.HasSuffix(file.Name, ".desc") { // Don't show for polygon description files
			zap.S().Warnf("Unrecognized submisison language for file %q", path.Base(file.Name))
		}
		return nil
	}

	ctx.submissions = append(ctx.submissions, &submissionStub{
		code: data,
		lang: lang,
	})
	return nil
}
