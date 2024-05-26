package tasks

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"go.uber.org/zap"
)

type ExecRequest struct {
	SubID     int
	SubtestID int
	Filename  string

	// TimeLimit is in seconds, MemoryLimit is in kilobytes
	MemoryLimit int
	TimeLimit   float64

	Lang   string
	TestID int
}

type ExecResponse struct {
	Time       float64
	Memory     int
	ExitStatus int
	Comments   string
}

func ExecuteTask(ctx context.Context, mgr eval.BoxScheduler, memQuota int64, req *ExecRequest, logger *slog.Logger) (*ExecResponse, error) {
	logger.Info("Executing subtest", slog.Int("subtest_id", req.SubtestID), slog.Int("sub_id", req.SubID))

	bucket, fileName := bucketFromIDExec(req.SubID)
	lang := eval.Langs[req.Lang]

	boxOut := fmt.Sprintf("/box/%s.out", req.Filename)

	bReq := &eval.Box2Request{
		InputBucketFiles: map[string]*eval.BucketFile{
			// Test input
			"/box/" + req.Filename + ".in": {
				Bucket:   datastore.BucketTypeTests,
				Filename: strconv.Itoa(req.TestID) + ".in",
				Mode:     0666,
			},
			// User executable
			lang.CompiledName: {
				Bucket:   bucket,
				Filename: fileName,
				Mode:     0777,
			},
		},

		RunConfig: &eval.RunConfig{
			EnvToSet:      maps.Clone(lang.RunEnv),
			MemoryLimit:   req.MemoryLimit,
			TimeLimit:     req.TimeLimit,
			WallTimeLimit: 2*req.TimeLimit + 1,
		},

		OutputBucketFiles: map[string]*eval.BucketFile{
			boxOut: {
				Bucket:   datastore.BucketTypeSubtests,
				Filename: strconv.Itoa(req.SubtestID),
				Mode:     0644,
			},
		},

		Command: slices.Clone(lang.RunCommand),
	}

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !lang.Compiled {
		bReq.RunConfig.Directories = slices.Clone(lang.Mounts)
	}

	if req.TimeLimit == 0 {
		bReq.RunConfig.WallTimeLimit = 30
	}

	if req.Filename == "stdin" {
		bReq.RunConfig.InputPath = "/box/stdin.in"
		bReq.RunConfig.OutputPath = "/box/stdin.out"
	}

	resp := &ExecResponse{}

	bResp, err := mgr.RunBox2(ctx, bReq, memQuota)
	if bResp == nil || err != nil {
		resp.Comments = "translate:internal_error"
		if err != nil {
			resp.Comments += "(" + err.Error() + ")"
		}
		return resp, nil
	}

	resp.Time, resp.Memory = bResp.Stats.Time, bResp.Stats.Memory

	okExit := false
	switch msg, status := bResp.Stats.Message, bResp.Stats.Status; status {
	case "TO":
		if strings.Contains(msg, "wall") {
			resp.Comments = "translate:walltimeout"
		} else {
			resp.Comments = "translate:timeout"
		}
	case "RE":
		resp.Comments = msg
	case "SG":
		resp.Comments = msg
	case "XX":
		resp.Comments = "Sandbox Error: " + msg
		zap.S().Warn("Sandbox error detected, check grader.log for more detials ", zap.Int("subtest_id", req.SubtestID), zap.Int("sub_id", req.SubID))
		logger.Warn("Sandbox error", slog.Int("sub_id", req.SubID), slog.Int("subtest_id", req.SubtestID), slog.Any("metadata", bResp.Stats))
	default:
		okExit = true
	}
	if !okExit {
		return resp, nil
	}

	if _, ok := bResp.BucketFiles[boxOut]; !ok {
		resp.Comments = "No output file found"
		return resp, nil
	}

	return resp, nil
}
