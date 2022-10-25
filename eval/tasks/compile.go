package tasks

import (
	"context"
	"fmt"
	"log"
	"os"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var _ eval.Task = &CompileTask{}

type CompileTask struct {
	Req   *eval.CompileRequest
	Resp  eval.CompileResponse
	Debug bool
}

func (job *CompileTask) Execute(ctx context.Context, box eval.Sandbox) error {
	if job.Debug {
		log.Printf("Compiling file using box %d\n", box.GetID())
	}

	lang, ok := eval.Langs[job.Req.Lang]
	if !ok {
		log.Printf("Language for submission %d could not be found: %q\n", job.Req.ID, job.Req.Lang)
		return kilonova.Statusf(500, "No language found")
	}

	outName := getIDExec(job.Req.ID)
	job.Resp.Success = true

	// If the language is interpreted, just save the code and leave
	if !lang.Compiled {
		// It should only be one file here anyway
		if len(job.Req.CodeFiles) > 1 {
			zap.S().Warn("More than one file specified for non-compiled language. This is not supported and will lead to strange behaviour")
		}
		for _, fData := range job.Req.CodeFiles {
			if err := os.WriteFile(outName, fData, 0644); err != nil {
				job.Resp.Other = err.Error()
				job.Resp.Success = false
			}
		}

		return nil
	}

	files := make(map[string][]byte)
	sourceFiles := []string{}
	for fName, fData := range job.Req.CodeFiles {
		files[fName] = fData
		sourceFiles = append(sourceFiles, fName)
	}
	for fName, fData := range job.Req.HeaderFiles {
		files[fName] = fData
	}

	out, err := eval.CompileFile(ctx, box, files, sourceFiles, lang)
	job.Resp.Output = out

	if err != nil {
		job.Resp.Success = false
	} else {
		f, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			job.Resp.Other = err.Error()
			job.Resp.Success = false
			return nil
		}
		if err := eval.CopyFromBox(box, lang.CompiledName, f); err != nil {
			job.Resp.Other = err.Error()
			job.Resp.Success = false
		}
		if err := f.Close(); err != nil {
			job.Resp.Other = err.Error()
			job.Resp.Success = false
		}
	}

	return nil
}

func getIDExec(id int) string {
	return path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", id))
}
