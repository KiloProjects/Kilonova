package tasks

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

type ExecRequest struct {
	SubID       int
	SubtestID   int
	Filename    string
	MemoryLimit int
	TimeLimit   float64
	Lang        string
	TestInput   io.Reader
}

type ExecResponse struct {
	Time       float64
	Memory     int
	ExitStatus int
	Comments   string
}

func GetExecuteTask(logger *zap.SugaredLogger) eval.Task[ExecRequest, ExecResponse] {
	return func(ctx context.Context, box eval.Sandbox, req *ExecRequest) (*ExecResponse, error) {
		resp := &ExecResponse{}
		logger.Infof("Executing test %d (for submission #%d) using box %d", req.SubtestID, req.SubID, box.GetID())

		if err := box.WriteFile("/box/"+req.Filename+".in", req.TestInput, 0644); err != nil {
			zap.S().Info("Can't write input file:", err)
			resp.Comments = "translate:internal_error"
			return resp, err
		}
		consoleInput := req.Filename == "stdin"

		lang := eval.Langs[req.Lang]
		if err := eval.CopyInBox(box, getIDExec(req.SubID), lang.CompiledName); err != nil {
			zap.S().Warn("Couldn't copy executable in box: ", err)
			resp.Comments = "translate:internal_error"
			return resp, err
		}

		meta, err := runSubmission(ctx, box, eval.Langs[req.Lang], req.TimeLimit, req.MemoryLimit, consoleInput)
		if err != nil {
			resp.Comments = fmt.Sprintf("Evaluation error: %v", err)
			return resp, nil
		}
		resp.Time = meta.Time
		resp.Memory = meta.Memory

		okExit := false
		switch meta.Status {
		case "TO":
			if strings.Contains(meta.Message, "wall") {
				resp.Comments = "translate:walltimeout"
			} else {
				resp.Comments = "translate:timeout"
			}
		case "RE":
			resp.Comments = meta.Message
		case "SG":
			resp.Comments = meta.Message
		case "XX":
			resp.Comments = "Sandbox Error: " + meta.Message
			zap.S().Warn("Sandbox error detected, check grader.log for more detials ", zap.Int("subtest_id", req.SubtestID), zap.Int("box_id", box.GetID()), zap.Int("sub_id", req.SubID))
			logger.Warn("Sandbox error: ", req.SubID, req.SubtestID, box.GetID(), spew.Sdump(meta))
		default:
			okExit = true
		}
		if !okExit {
			return resp, nil
		}

		boxOut := fmt.Sprintf("/box/%s.out", req.Filename)
		if !box.FileExists(boxOut) {
			resp.Comments = "No output file found"
			return resp, nil
		}

		w, err := datastore.GetBucket(datastore.BucketTypeSubtests).Writer(strconv.Itoa(req.SubtestID), 0644)
		if err != nil {
			resp.Comments = "Could not open problem output"
			return resp, nil
		}

		if err := box.ReadFile(boxOut, w); err != nil {
			resp.Comments = "Could not write output file"
			return resp, nil
		}

		if err := w.Close(); err != nil {
			zap.S().Warn(err)
			resp.Comments = "Could not close output file"
			return resp, nil
		}

		return resp, nil
	}
}

// runSubmission runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
// timeLimit is in seconds, memoryLimit is in kilbytes
func runSubmission(ctx context.Context, box eval.Sandbox, language eval.Language, timeLimit float64, memoryLimit int, consoleInput bool) (*eval.RunStats, error) {

	var runConf eval.RunConfig
	runConf.EnvToSet = make(map[string]string)

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.Compiled {
		runConf.Directories = append(runConf.Directories, language.Mounts...)
	}

	for key, val := range language.RunEnv {
		runConf.EnvToSet[key] = val
	}

	runConf.MemoryLimit = memoryLimit
	runConf.TimeLimit = timeLimit
	runConf.WallTimeLimit = 2*timeLimit + 1
	if timeLimit == 0 {
		runConf.WallTimeLimit = 30
	}

	if consoleInput {
		runConf.InputPath = "/box/stdin.in"
		runConf.OutputPath = "/box/stdin.out"
	}

	goodCmd, err := eval.MakeGoodCommand(language.RunCommand)
	if err != nil {
		zap.S().Warnf("MakeGoodCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.RunCommand)
		goodCmd = language.RunCommand
	}

	return box.RunCommand(ctx, goodCmd, &runConf)
}
