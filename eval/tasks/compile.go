package tasks

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
)

const outputPath = "/box/compilation.out"

type CompileRequest struct {
	// TODO: Better identifier for such requests
	ID          int
	CodeFiles   map[string][]byte
	HeaderFiles map[string][]byte
	Lang        *eval.Language
}

type CompileResponse struct {
	Output  string
	Success bool
	Other   string

	Stats *eval.RunStats
}

const compileOutputLimit = 4500 // runes

// returns the filename to save with and the bucket to save into
func bucketFromIDExec(id int) (datastore.BucketType, string) {
	if id < 0 { // checker
		// use -id to turn back positive
		return datastore.BucketTypeCheckers, fmt.Sprintf("%d.bin", -id)
	}
	return datastore.BucketTypeCompiles, fmt.Sprintf("%d.bin", id)
}

func CompileTask(ctx context.Context, mgr eval.BoxScheduler, req *CompileRequest, logger *slog.Logger) (*CompileResponse, error) {
	resp := &CompileResponse{}

	// TODO: I don't think we need this anymore
	if req.Lang == nil {
		slog.Warn("Could not find language for submission", slog.Any("sub_id", req.ID), slog.Any("lang", req.Lang))
		return resp, kilonova.Statusf(500, "No language found")
	}

	bucket, outName := bucketFromIDExec(req.ID)
	resp.Success = true

	// If the language is interpreted, just save the code and leave
	if !req.Lang.Compiled {
		// It should only be one file here anyway
		if len(req.CodeFiles) > 1 {
			slog.Warn("More than one file specified for non-compiled language. This is not properly supported")
		}
		for _, fData := range req.CodeFiles {
			if err := datastore.GetBucket(bucket).WriteFile(outName, bytes.NewBuffer(fData), 0644); err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
		}
		return resp, nil
	}

	logger.Info("Compiling file", slog.Int("req_id", req.ID))

	bReq := &eval.Box2Request{
		InputByteFiles: make(map[string]*eval.ByteFile),

		// Compilation output
		OutputBucketFiles: map[string]*eval.BucketFile{
			req.Lang.CompiledName: {
				Bucket:   bucket,
				Filename: outName,
				Mode:     0777,
			},
		},
		OutputByteFiles: []string{outputPath},

		// Run config
		RunConfig: &eval.RunConfig{
			EnvToSet:    maps.Clone(req.Lang.BuildEnv),
			InheritEnv:  true,
			Directories: slices.Clone(req.Lang.Mounts),

			StderrToStdout: true,
			OutputPath:     outputPath,
		},
	}

	// File environment
	sourceFiles := []string{}
	for fName, fData := range req.CodeFiles {
		bReq.InputByteFiles[fName] = &eval.ByteFile{
			Data: fData,
			Mode: 0666,
		}
		sourceFiles = append(sourceFiles, fName)
	}
	for fName, fData := range req.HeaderFiles {
		bReq.InputByteFiles[fName] = &eval.ByteFile{
			Data: fData,
			Mode: 0666,
		}
	}

	// Init compilation command
	goodCmd, err := makeGoodCompileCommand(req.Lang.CompileCommand, sourceFiles)
	if err != nil {
		slog.Warn("error in makeGoodCompileCommand. This is not good, so we'll use the command from the config file", slog.Any("err", err), slog.Any("command", req.Lang.CompileCommand))
		goodCmd = req.Lang.CompileCommand
	}
	bReq.Command = goodCmd

	// TODO: Maybe define a max memory quota for compilations?
	bResp, err := mgr.RunBox2(ctx, bReq, 0)
	if bResp == nil {
		resp.Output = "Internal runner error"
		resp.Success = false
		return resp, nil
	}
	resp.Output = compilationOutput(bResp)
	resp.Stats = bResp.Stats

	if err != nil {
		resp.Success = false
		return resp, nil
	}

	if bResp.Stats.ExitCode > 0 {
		resp.Success = false
		return resp, nil
	}

	if _, ok := bResp.BucketFiles[req.Lang.CompiledName]; !ok {
		resp.Other = "Could not save compilation output"
		resp.Success = false
	}

	return resp, nil
}

func compilationOutput(resp *eval.Box2Response) string {
	if resp == nil {
		return ""
	}
	val, ok := resp.ByteFiles[outputPath]
	if !ok {
		return ""
	}
	combinedOutRunes := []rune(string(val))

	if len(combinedOutRunes) > compileOutputLimit { // Truncate output on error
		combinedOutRunes = append(combinedOutRunes[:compileOutputLimit], []rune("... (compilation output trimmed)")...)
	}
	return string(combinedOutRunes)
}

func makeGoodCompileCommand(command []string, files []string) ([]string, error) {
	cmd := slices.Clone(command)
	for i := range cmd {
		if cmd[i] == eval.MagicReplace {
			x := []string{}
			x = append(x, cmd[:i]...)
			x = append(x, files...)
			x = append(x, cmd[i+1:]...)
			return x, nil
		}
	}

	slog.Warn("Did not replace any fields in command", slog.Any("command", command))
	return cmd, nil
}
