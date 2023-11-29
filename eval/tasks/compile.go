package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

const compileOutputLimit = 4500 // runes

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

		out, stats, err := compileFile(ctx, box, files, sourceFiles, lang)
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

// compileFile compiles a file that has the corresponding language
func compileFile(ctx context.Context, box eval.Sandbox, files map[string][]byte, compiledFiles []string, language eval.Language) (string, *eval.RunStats, error) {
	for fileName, fileData := range files {
		if err := box.WriteFile(fileName, bytes.NewReader(fileData), 0644); err != nil {
			return "", nil, err
		}
	}

	var conf eval.RunConfig
	conf.EnvToSet = make(map[string]string)

	conf.InheritEnv = true
	conf.Directories = append(conf.Directories, language.Mounts...)

	for key, val := range language.BuildEnv {
		conf.EnvToSet[key] = val
	}

	conf.StderrToStdout = true
	conf.OutputPath = "/box/compilation.out"

	goodCmd, err := makeGoodCompileCommand(language.CompileCommand, compiledFiles)
	if err != nil {
		zap.S().Warnf("MakeGoodCompileCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.CompileCommand)
		goodCmd = language.CompileCommand
	}

	stats, err := box.RunCommand(ctx, goodCmd, &conf)

	var out bytes.Buffer
	if err := box.ReadFile("/box/compilation.out", &out); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			zap.S().Warn("Couldn't read compilation output: ", err)
			out.Reset()
		}
	}

	combinedOutRunes := []rune(out.String())

	if len(combinedOutRunes) > compileOutputLimit { // Truncate output on error
		combinedOutRunes = append(combinedOutRunes[:compileOutputLimit], []rune("... (compilation output trimmed)")...)
	}
	combinedOut := string(combinedOutRunes)

	if err != nil {
		return combinedOut, stats, err
	}

	return combinedOut, stats, nil
}

func getIDExec(id int) string {
	if id < 0 { // checker
		// use -id to turn back positive
		return path.Join(config.Eval.CompilePath, "checker_cache", fmt.Sprintf("%d.bin", -id))
	}
	return path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", id))
}

func makeGoodCompileCommand(command []string, files []string) ([]string, error) {
	cmd, err := eval.MakeGoodCommand(command)
	if err != nil {
		return nil, err
	}
	for i := range cmd {
		if cmd[i] == eval.MAGIC_REPLACE {
			x := []string{}
			x = append(x, cmd[:i]...)
			x = append(x, files...)
			x = append(x, cmd[i+1:]...)
			return x, nil
		}
	}

	zap.S().Warnf("Didn't replace any fields in %#v", command)
	return cmd, nil
}
