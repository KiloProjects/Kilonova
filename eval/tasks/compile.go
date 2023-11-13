package tasks

import (
	"context"
	"fmt"
	"os"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

func GetCompileTask(logger *zap.SugaredLogger) eval.Task[eval.CompileRequest, eval.CompileResponse] {
	return func(ctx context.Context, box eval.Sandbox, req *eval.CompileRequest) (*eval.CompileResponse, error) {
		resp := &eval.CompileResponse{}
		logger.Infof("Compiling file using box %d", box.GetID())

		lang, ok := eval.Langs[req.Lang]
		if !ok {
			zap.S().Warnf("Language for submission %d could not be found: %q", req.ID, req.Lang)
			return resp, kilonova.Statusf(500, "No language found")
		}

		outName := getIDExec(req.ID)
		resp.Success = true

		// If the language is interpreted, just save the code and leave
		if !lang.Compiled {
			// It should only be one file here anyway
			if len(req.CodeFiles) > 1 {
				zap.S().Warn("More than one file specified for non-compiled language. This is not supported")
			}
			for _, fData := range req.CodeFiles {
				if err := os.WriteFile(outName, fData, 0644); err != nil {
					resp.Other = err.Error()
					resp.Success = false
				}
			}
			return resp, nil
		}

		files := make(map[string][]byte)
		sourceFiles := []string{}
		for fName, fData := range req.CodeFiles {
			files[fName] = fData
			sourceFiles = append(sourceFiles, fName)
		}
		for fName, fData := range req.HeaderFiles {
			files[fName] = fData
		}

		out, stats, err := eval.CompileFile(ctx, box, files, sourceFiles, lang)
		resp.Output = out
		resp.Stats = stats

		if err != nil {
			resp.Success = false
			return resp, nil
		}

		f, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
		if err != nil {
			resp.Other = err.Error()
			resp.Success = false
			return resp, nil
		}
		if err := box.ReadFile(lang.CompiledName, f); err != nil {
			resp.Other = err.Error()
			resp.Success = false
		}
		if err := f.Close(); err != nil {
			resp.Other = err.Error()
			resp.Success = false
		}

		return resp, nil
	}
}

func getIDExec(id int) string {
	if id < 0 { // checker
		// use -id to turn back positive
		return path.Join(config.Eval.CompilePath, "checker_cache", fmt.Sprintf("%d.bin", -id))
	}
	return path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", id))
}
