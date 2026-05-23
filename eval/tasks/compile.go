package tasks

import (
	"bytes"
	"context"
	"errors"
	"log/slog"

	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/davecgh/go-spew/spew"
)

const outputPath = "/box/compilation.out"

type CompileRequest struct {
	File        *eval.BucketFile
	CodeFiles   map[string][]byte
	HeaderFiles map[string][]byte
	Lang        language.GraderLang
	Store       *datastore.Manager

	OriginalFilename string
}

type CompileResponse struct {
	Output  string
	Success bool
	Other   string

	Stats *eval.RunStats
}

const compileOutputLimit = 4500 // runes

func CompileTask(ctx context.Context, mgr eval.BoxScheduler, req *CompileRequest, logger *slog.Logger) (*CompileResponse, error) {
	resp := &CompileResponse{}

	// TODO: I don't think we need this anymore
	if req.Lang == nil {
		slog.WarnContext(ctx, "Could not find language for submission", slog.Any("sub_file", req.File), slog.Any("lang", req.Lang))
		return resp, errors.New("no language found")
	}

	if req.File == nil {
		slog.WarnContext(ctx, "Could not find file for submission")
		return resp, errors.New("no file found")
	}

	resp.Success = true

	// If the language is interpreted, just save the code and leave
	if !req.Lang.Compiled() {
		// It should only be one file here anyway
		if len(req.CodeFiles) > 1 {
			slog.WarnContext(ctx, "More than one file specified for non-compiled language. This is not properly supported")
		}
		for _, fData := range req.CodeFiles {
			b, err := req.Store.Get(req.File.Bucket)
			if err != nil {
				resp.Other = err.Error()
				resp.Success = false
			} else if err := b.WriteFile(req.File.Filename, bytes.NewBuffer(fData), 0644); err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
		}
		return resp, nil
	}

	logger.InfoContext(ctx, "Compiling file", slog.Any("req_file", req.File))

	bReq := &eval.Box2Request{
		InputByteFiles: make(map[string]*eval.ByteFile),

		// Compilation output
		OutputBucketFiles: map[string]*eval.BucketFile{
			req.Lang.CompiledName(req.OriginalFilename): req.File,
		},
		OutputByteFiles: []string{outputPath},

		// Run config
		RunConfig: &eval.RunConfig{
			EnvToSet:    req.Lang.BuildEnv(),
			InheritEnv:  true,
			Directories: req.Lang.Mounts(),

			TimeLimit:     20,         // 20 seconds
			WallTimeLimit: 30,         // 30 seconds
			MemoryLimit:   512 * 1024, // 1024MB

			StderrToStdout: true,
			OutputPath:     outputPath,

			EnableInternet: true,
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
	bReq.Command = req.Lang.CompileCommand(sourceFiles)

	// TODO: Maybe define a max memory quota for compilations?
	bResp, err := mgr.RunBox2(ctx, bReq, 0)
	if bResp == nil {
		spew.Dump(bReq)
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

	if _, ok := bResp.BucketFiles[req.Lang.CompiledName(req.OriginalFilename)]; !ok {
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
