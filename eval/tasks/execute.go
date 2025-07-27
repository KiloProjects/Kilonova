package tasks

import (
	"cmp"
	"context"
	"errors"
	"log/slog"
	"maps"
	"math"
	"slices"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/shopspring/decimal"
)

const (
	managerMemoryLimit = 512 * 1024
	managerTimeLimit   = 20
)

type BatchResponse struct {
	Time     float64
	Memory   int
	Comments string
}

type BatchRequest struct {
	SubID      int
	SubtestID  int
	InputName  string
	OutputName string

	// TimeLimit is in seconds, MemoryLimit is in kilobytes
	MemoryLimit int
	TimeLimit   float64

	Lang   *eval.Language
	TestID int
}

func ExecuteBatch(ctx context.Context, mgr eval.BoxScheduler, memQuota int64, req *BatchRequest, logger *slog.Logger) (*BatchResponse, error) {
	logger.InfoContext(ctx, "Executing batch subtest", slog.Int("subtest_id", req.SubtestID), slog.Int("sub_id", req.SubID))

	bucketFile := bucketFileFromID(req.SubID, 0777)

	bReq := &eval.Box2Request{
		InputBucketFiles: map[string]*eval.BucketFile{
			// Test input
			"/box/" + req.InputName: {
				Bucket:   datastore.BucketTypeTests,
				Filename: strconv.Itoa(req.TestID) + ".in",
				Mode:     0666,
			},
			// User executable
			req.Lang.CompiledName: bucketFile,
		},

		RunConfig: &eval.RunConfig{
			EnvToSet:      maps.Clone(req.Lang.RunEnv),
			MemoryLimit:   req.MemoryLimit,
			TimeLimit:     req.TimeLimit,
			WallTimeLimit: 2*req.TimeLimit + 1,
		},

		OutputBucketFiles: map[string]*eval.BucketFile{
			"/box/" + req.OutputName: {
				Bucket:   datastore.BucketTypeSubtests,
				Filename: strconv.Itoa(req.SubtestID),
				Mode:     0644,
			},
		},

		Command: slices.Clone(req.Lang.RunCommand),
	}

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !req.Lang.Compiled {
		bReq.RunConfig.Directories = slices.Clone(req.Lang.Mounts)
	}

	if req.TimeLimit == 0 {
		bReq.RunConfig.WallTimeLimit = 30
	}

	if req.InputName == "stdin" {
		bReq.RunConfig.InputPath = "/box/stdin"
	}
	if req.OutputName == "stdout" {
		bReq.RunConfig.OutputPath = "/box/stdout"
	}

	bResp, err := mgr.RunBox2(ctx, bReq, memQuota)
	if bResp == nil || err != nil {
		resp := &BatchResponse{}
		resp.Comments = "translate:internal_error"
		if err != nil {
			resp.Comments += "(" + err.Error() + ")"
		}
		return resp, nil
	}

	resp := parseResponse(ctx, bResp.Stats, logger, req.SubtestID)

	if _, ok := bResp.BucketFiles["/box/"+req.OutputName]; !ok {
		resp.Comments = "No output file found"
	}

	return &resp, nil
}

type CommunicationRequest struct {
	SubID     int
	SubtestID int
	UseStdin  bool

	// TimeLimit is in seconds, MemoryLimit is in kilobytes
	MemoryLimit int
	TimeLimit   float64

	SubLang     *eval.Language
	CheckerLang *eval.Language
	TestID      int

	NumUserSandboxes int64
}

type CommunicationResponse struct {
	BatchResponse

	Score decimal.Decimal
}

func ExecuteCommunication(ctx context.Context, mgr eval.BoxScheduler, cName string, memQuota int64, req *CommunicationRequest, logger *slog.Logger) (*CommunicationResponse, error) {
	logger.InfoContext(ctx, "Executing communication subtest", slog.Int("subtest_id", req.SubtestID), slog.Int("sub_id", req.SubID))

	if cName == "" {
		return nil, errors.New("no checker name provided")
	}

	managerBucketExec := bucketFileFromID(-req.SubID, 0777)
	userBucketExec := bucketFileFromID(req.SubID, 0777)

	managerReq := &eval.Box2Request{
		InputBucketFiles: map[string]*eval.BucketFile{
			// Test input
			"/box/input.txt": {
				Bucket:   datastore.BucketTypeTests,
				Filename: strconv.Itoa(req.TestID) + ".in",
				Mode:     0666,
			},
			// Manager executable
			req.CheckerLang.CompiledName: managerBucketExec,
		},

		Command: slices.Clone(req.CheckerLang.RunCommand),
		RunConfig: &eval.RunConfig{
			InputPath:  "/box/input.txt",
			OutputPath: "/box/verdict.out",
			StderrPath: "/box/verdict.err",

			MemoryLimit: managerMemoryLimit,
			TimeLimit:   max(managerTimeLimit, float64(req.NumUserSandboxes)*(req.TimeLimit+1.0)),
		},

		OutputByteFiles: []string{"/box/verdict.out", "/box/verdict.err"},
	}

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !req.CheckerLang.Compiled {
		managerReq.RunConfig.Directories = slices.Clone(req.CheckerLang.Mounts)
	}

	userReqs := make([]*eval.Box2Request, req.NumUserSandboxes)

	for i := range userReqs {
		userReqs[i] = &eval.Box2Request{
			InputBucketFiles: map[string]*eval.BucketFile{
				// User executable
				req.SubLang.CompiledName: userBucketExec,
			},

			RunConfig: &eval.RunConfig{
				EnvToSet:      maps.Clone(req.SubLang.RunEnv),
				MemoryLimit:   req.MemoryLimit,
				TimeLimit:     req.TimeLimit,
				WallTimeLimit: 2*req.TimeLimit + 1,
			},

			Command: slices.Clone(req.SubLang.RunCommand),
		}

		// if our specified language is not compiled, then it means that
		// the mounts specified should be added at runtime
		if !req.SubLang.Compiled {
			userReqs[i].RunConfig.Directories = slices.Clone(req.SubLang.Mounts)
		}

		if req.TimeLimit == 0 {
			userReqs[i].RunConfig.WallTimeLimit = 30
		}
	}

	bResp, userStats, err := mgr.RunMultibox(ctx, &eval.MultiboxRequest{
		ManagerSandbox:     managerReq,
		UserSandboxConfigs: userReqs,

		UseStdin: req.UseStdin,
	}, managerMemoryLimit, memQuota)
	if bResp == nil || err != nil {
		resp := &CommunicationResponse{}
		resp.Comments = "translate:internal_error"
		if err != nil {
			resp.Comments += "(" + err.Error() + ")"
		}
		return resp, nil
	}
	userResp := parseResponse(ctx, mergeStats(true, userStats...), logger, req.SubtestID)
	mgrResp := parseResponse(ctx, bResp.Stats, logger, req.SubtestID)
	mgrResp.Time = userResp.Time
	mgrResp.Memory = userResp.Memory

	if userResp.Comments != "" {
		return &CommunicationResponse{
			BatchResponse: userResp,
			Score:         decimal.Zero,
		}, nil
	}

	if mgrResp.Comments != "" {
		return &CommunicationResponse{
			BatchResponse: mgrResp,
			Score:         decimal.Zero,
		}, nil
	}

	score, output := ParseStandardManagerOutput(bResp)
	mgrResp.Comments = output

	return &CommunicationResponse{
		BatchResponse: mgrResp,
		Score:         score,
	}, nil
}

func parseResponse(ctx context.Context, stats *eval.RunStats, logger *slog.Logger, subtestID int) BatchResponse {
	resp := BatchResponse{}

	resp.Time, resp.Memory = stats.Time, stats.Memory

	okExit := false
	switch msg, status := stats.Message, stats.Status; status {
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
		slog.WarnContext(ctx, "Sandbox error detected, check grader.log for more details", slog.Int("subtest_id", subtestID))
		logger.WarnContext(ctx, "Sandbox error", slog.Int("subtest_id", subtestID), slog.Any("metadata", stats))
	default:
		okExit = true
	}
	if !okExit {
		return resp
	}

	if stats.MemoryLimitExceeded {
		resp.Comments = "translate:memory_limit"
	}

	return resp
}

func mergeStats(concurrent bool, stats ...*eval.RunStats) *eval.RunStats {
	var mergedStats eval.RunStats

	for _, stat := range stats {
		if concurrent {
			mergedStats.Memory += stat.Memory
			mergedStats.Time = max(mergedStats.Time, stat.Time)
		} else {
			mergedStats.Memory = max(mergedStats.Memory, stat.Memory)
			mergedStats.Time += stat.Time
		}
		stat.Killed = cmp.Or(stat.Killed, mergedStats.Killed)
		stat.ExitCode = cmp.Or(stat.ExitCode, mergedStats.ExitCode)
		stat.ExitSignal = cmp.Or(stat.ExitSignal, mergedStats.ExitSignal)
		stat.Message = cmp.Or(stat.Message, mergedStats.Message)
		stat.Status = cmp.Or(stat.Status, mergedStats.Status)
		stat.MemoryLimitExceeded = cmp.Or(stat.MemoryLimitExceeded, mergedStats.MemoryLimitExceeded)
	}

	return &mergedStats
}

func ParseStandardManagerOutput(resp *eval.Box2Response) (decimal.Decimal, string) {
	stdout, ok := resp.ByteFiles["/box/verdict.out"]
	if !ok {
		stdout = []byte{}
	}
	stderr, ok := resp.ByteFiles["/box/verdict.err"]
	if !ok {
		stderr = []byte{}
	}

	floatScore, err := strconv.ParseFloat(strings.TrimSpace(string(stdout)), 64)
	if err != nil || math.IsInf(floatScore, 0) || math.IsNaN(floatScore) {
		return decimal.Zero, "Invalid score"
	}

	output := strings.TrimSpace(string(stderr))
	if output == "" {
		output = "No message"
	}

	return decimal.NewFromFloat(floatScore).Shift(2), output
}
