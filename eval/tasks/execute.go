package tasks

import (
	"context"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
)

func GetExecuteTask(logger *zap.SugaredLogger, dm kilonova.GraderStore) eval.Task[eval.ExecRequest, eval.ExecResponse] {
	return func(ctx context.Context, box eval.Sandbox, req *eval.ExecRequest) (*eval.ExecResponse, error) {
		resp := &eval.ExecResponse{}
		logger.Infof("Executing test %d (for submission #%d) using box %d", req.SubtestID, req.SubID, box.GetID())

		{
			dir, err := box.ReadDir("/box/")
			if err != nil {
				zap.S().Warn("/box/ read error, check grader.log for details")
				logger.Infof("Can't read /box/: ", err)
			} else if len(dir) > 0 {
				zap.S().Warn("/box/ anomaly, check grader.log for details")
				logger.Warnf("Box %d (test %d, submission %d) directory is not initially empty: %#v", box.GetID(), req.SubtestID, req.SubID, dir)
			}
		}

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

		lim := eval.Limits{
			MemoryLimit: req.MemoryLimit,
			TimeLimit:   req.TimeLimit,
		}
		meta, err := eval.RunSubmission(ctx, box, eval.Langs[req.Lang], lim, consoleInput)
		if err != nil {
			resp.Comments = fmt.Sprintf("Error running submission: %v", err)
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
			zap.S().Warn("Sandbox eror detected, check grader.log for more detials ", zap.Int("subtest_id", req.SubtestID), zap.Int("box_id", box.GetID()), zap.Int("sub_id", req.SubID))
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
			zap.S().Warn("No output file found, check grader.log for more details ", zap.Int("subtest_id", req.SubtestID), zap.Int("box_id", box.GetID()), zap.Int("sub_id", req.SubID))
			logger.Info("This may be a bug: ", spew.Sdump(box.ReadDir("/box/")), spew.Sdump(meta))
			return resp, nil
		}

		w, err := dm.SubtestWriter(req.SubtestID)
		if err != nil {
			resp.Comments = "Could not open problem output"
			return resp, nil
		}

		if err := box.ReadFile(boxOut, w); err != nil {
			resp.Comments = "Could not write output file"
			return resp, nil
		}

		if err := w.Close(); err != nil {
			resp.Comments = "Could not close output file"
			return resp, nil
		}

		return resp, nil
	}
}
