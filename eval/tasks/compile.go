package tasks

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
)

type CompileRequest struct {
	// TODO: Better identifier for such requests
	ID          int
	CodeFiles   map[string][]byte
	HeaderFiles map[string][]byte
	Lang        string
}

type CompileResponse struct {
	Output  string
	Success bool
	Other   string

	Stats *eval.RunStats
}

const compileOutputLimit = 4500 // runes

// returns the filename to save with and the bucket to save into
func bucketFromIDExec(id int) (*datastore.Bucket, string) {
	if id < 0 { // checker
		// use -id to turn back positive
		return datastore.GetBucket(datastore.BucketTypeCheckers), fmt.Sprintf("%d.bin", -id)
	}
	return datastore.GetBucket(datastore.BucketTypeCompiles), fmt.Sprintf("%d.bin", id)
}

func GetCompileTask(logger *zap.SugaredLogger) eval.Task[CompileRequest, CompileResponse] {
	return func(ctx context.Context, box eval.Sandbox, req *CompileRequest) (*CompileResponse, error) {
		resp := &CompileResponse{}
		logger.Infof("Compiling file using box %d", box.GetID())

		lang, ok := eval.Langs[req.Lang]
		if !ok {
			zap.S().Warnf("Language for submission %d could not be found: %q", req.ID, req.Lang)
			return resp, kilonova.Statusf(500, "No language found")
		}

		bucket, outName := bucketFromIDExec(req.ID)
		resp.Success = true

		// If the language is interpreted, just save the code and leave
		if !lang.Compiled {
			// It should only be one file here anyway
			if len(req.CodeFiles) > 1 {
				zap.S().Warn("More than one file specified for non-compiled language. This is not supported")
			}
			for _, fData := range req.CodeFiles {
				if err := bucket.WriteFile(outName, bytes.NewBuffer(fData), 0644); err != nil {
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

		pr, pw := io.Pipe()
		go func() {
			err := box.ReadFile(lang.CompiledName, pw)
			if err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
			pw.Close()
		}()

		if err := bucket.WriteFile(outName, pr, 0777); err != nil {
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
			zap.S().Warn(err)
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

func makeGoodCompileCommand(command []string, files []string) ([]string, error) {
	cmd, err := eval.MakeGoodCommand(command)
	if err != nil {
		return nil, err
	}
	for i := range cmd {
		if cmd[i] == eval.MagicReplace {
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
