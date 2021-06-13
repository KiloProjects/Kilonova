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
		log.Printf("Language for submission %d could not be found\n", job.Req.ID)
		return &kilonova.Error{Code: kilonova.EINTERNAL, Message: "No language found"}
	}

	outName := getIDExec(job.Req.ID)
	job.Resp.Success = true

	if lang.Compiled {
		out, err := eval.CompileFile(ctx, box, job.Req.Code, lang)
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

	if err := os.WriteFile(outName, job.Req.Code, 0644); err != nil {
		job.Resp.Other = err.Error()
		job.Resp.Success = false
	}

	return nil
}

func getIDExec(id int) string {
	return path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", id))
}
