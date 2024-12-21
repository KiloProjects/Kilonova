package grader

import (
	"context"
	"errors"
	"fmt"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"log/slog"
	"os"
	"path"
	"slices"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/box"
	"github.com/KiloProjects/kilonova/eval/checkers"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/tasks"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/davecgh/go-spew/spew"
	"github.com/shopspring/decimal"
)

var (
	True            = true
	skippedVerdict  = "translate:skipped"
	acceptedVerdict = "test_verdict.accepted"
)

type submissionHandler struct {
	base *sudoapi.BaseAPI

	runner eval.BoxScheduler

	settings *kilonova.ProblemEvalSettings
	pb       *kilonova.Problem
	sub      *kilonova.Submission

	lang *eval.Language
}

var GraphvizSave = config.GenFlag("experimental.grader.save_graphviz", false, "Save graphviz .dot files to tmp directory for run graph debugging purposes")

func (sh *submissionHandler) buildRunGraph(ctx context.Context, subtests []*kilonova.SubTest) (graph.Graph[int, *kilonova.SubTest], error) {
	g := graph.New(func(sub *kilonova.SubTest) int {
		if sub == nil {
			return -1
		}
		return sub.ID
	}, graph.Acyclic(), graph.PreventCycles(), graph.Directed(), graph.Rooted())
	_ = g.AddVertex(nil)
	for _, test := range subtests {
		if err := g.AddVertex(test); err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return nil, err
		}
	}
	switch sh.sub.SubmissionType {
	case kilonova.EvalTypeClassic:
		stks, err := sh.base.SubmissionSubTasks(ctx, sh.sub.ID)
		if err != nil {
			return nil, err
		}
		if len(stks) == 0 {
			for _, subtest := range subtests {
				if err := g.AddEdge(-1, subtest.ID); err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
					return nil, err
				}
			}
		} else {
			for _, subtask := range stks {
				// Only do it if sequential
				//if subtask.Ordering = kilonova.SubtaskOrderingSequential {
				lastVertex := -1
				for i := range subtask.Subtests {
					if err := g.AddEdge(lastVertex, subtask.Subtests[i]); err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
						return nil, err
					}
					lastVertex = subtask.Subtests[i]
				}
				//}
			}
		}
	case kilonova.EvalTypeICPC:
		lastsVertex := -1
		for i := range subtests {
			if err := g.AddEdge(lastsVertex, subtests[i].ID); err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
				return nil, err
			}
			lastsVertex = subtests[i].ID
		}
	}

	edges, err := g.Edges()
	if err != nil {
		return nil, err
	}
	satisfiedSubtests := make(map[int]bool)
	for _, edge := range edges {
		satisfiedSubtests[edge.Target] = true
	}
	for _, subtest := range subtests {
		if _, ok := satisfiedSubtests[subtest.ID]; !ok {
			_ = g.AddEdge(-1, subtest.ID)
		}
	}

	return g, nil
}

func (sh *submissionHandler) genSubCompileRequest(ctx context.Context) (*tasks.CompileRequest, error) {
	req := &tasks.CompileRequest{
		ID:          sh.sub.ID,
		Lang:        sh.lang,
		CodeFiles:   make(map[string][]byte),
		HeaderFiles: make(map[string][]byte),
	}
	atts, err := sh.base.ProblemAttachments(ctx, sh.pb.ID)
	if err != nil {
		return nil, err
	}

	for _, codeFile := range sh.settings.GraderFiles {
		lang := sh.runner.LanguageFromFilename(codeFile)
		if lang == nil || (lang.InternalName != sh.lang.InternalName && !slices.Contains(sh.lang.SimilarLangs, lang.InternalName)) {
			continue
		}
		for _, att := range atts {
			if att.Name == codeFile {
				data, err := sh.base.AttachmentData(ctx, att.ID)
				if err != nil {
					slog.WarnContext(ctx, "Couldn't get attachment data", slog.Any("err", err))
					return nil, kilonova.Statusf(500, "Couldn't get grader data")
				}
				name := strings.Replace(path.Base(att.Name), path.Ext(att.Name), lang.Extensions[0], 1)
				req.CodeFiles[path.Join("/box", name)] = data
			}
		}
	}
	subCode, err := sh.base.RawSubmissionCode(ctx, sh.sub.ID)
	if err != nil {
		return nil, err
	}
	if len(sh.settings.GraderFiles) > 0 && sh.sub.Language == "pascal" {
		// In interactive problems, include the source code as header
		// Apparently the fpc compiler allows only one file as parameter, this should solve it
		req.HeaderFiles[sh.lang.SourceName] = subCode
	} else {
		// But by default it should be a code file
		req.CodeFiles[sh.lang.SourceName] = subCode
	}
	for _, headerFile := range sh.settings.HeaderFiles {
		for _, att := range atts {
			if att.Name == headerFile {
				data, err := sh.base.AttachmentData(ctx, att.ID)
				if err != nil {
					slog.WarnContext(ctx, "Couldn't get attachment data", slog.Any("err", err))
					return nil, kilonova.Statusf(500, "Couldn't get grader data")
				}
				req.HeaderFiles[path.Join("/box", path.Base(att.Name))] = data
			}
		}
	}
	return req, nil
}

func executeSubmission(ctx context.Context, base *sudoapi.BaseAPI, runner eval.BoxScheduler, sub *kilonova.Submission) error {
	graderLogger.InfoContext(ctx, "Executing submission", slog.Int("id", sub.ID), slog.Any("status", sub.Status))
	defer func() {
		// In case anything ever happens, make sure it is at least marked as finished
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished}); err != nil {
			slog.WarnContext(ctx, "Couldn't finish submission", slog.Any("err", err))
		}
	}()

	sh := submissionHandler{
		base:   base,
		runner: runner,
		sub:    sub,
		lang:   runner.Language(sub.Language),
	}
	if sh.lang == nil {
		slog.WarnContext(ctx, "Could not find submission language when evaluating", slog.String("lang", sub.Language))
		return kilonova.Statusf(500, "Language not found for submission")
	}

	defer func() {
		err := sh.markSubtestsDone(ctx)
		if err != nil {
			slog.WarnContext(ctx, "Couldn't clean up subtests", slog.Any("err", err))
		}
	}()

	problem, err1 := base.Problem(ctx, sub.ProblemID)
	if err1 != nil {
		return fmt.Errorf("Couldn't get submission problem: %w", err1)
	}

	problemSettings, err1 := base.ProblemSettings(ctx, sub.ProblemID)
	if err1 != nil {
		return fmt.Errorf("Couldn't get problem settings: %w", err1)
	}

	sh.pb = problem
	sh.settings = problemSettings

	if err := sh.compileSubmission(ctx); err != nil {
		if kilonova.ErrorCode(err) != 204 { // Skip
			slog.WarnContext(ctx, "Non-skip error code", slog.Any("err", err))
			return err
		}
		return nil
	}

	checker, err := sh.getAppropriateChecker(ctx)
	if err != nil {
		return fmt.Errorf("Couldn't get checker: %w", err)
	}

	if info, err := checker.Prepare(ctx); err != nil {
		t := true
		info = "Checker compile error:\n" + info
		internalErr := "test_verdict.internal_error"
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &problem.DefaultPoints,
			CompileError: &t, CompileMessage: &info,
			ChangeVerdict: true, ICPCVerdict: &internalErr,
		}); err != nil {
			return fmt.Errorf("Error during update of compile information: %w", err)
		}
		return fmt.Errorf("Could not prepare checker: %w", err)
	}

	subTests, err1 := base.SubTests(ctx, sub.ID)
	if err1 != nil {
		internalErr := "test_verdict.internal_error"
		if err := base.UpdateSubmission(ctx, sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &problem.DefaultPoints,
			ChangeVerdict: true, ICPCVerdict: &internalErr,
		}); err != nil {
			return fmt.Errorf("Could not update submission after subtest fetch fail: %w", err)
		}
		return fmt.Errorf("Could not fetch subtests: %w", err1)
	}

	if g, err := sh.buildRunGraph(ctx, subTests); err != nil {
		slog.WarnContext(ctx, "Error building experimental run graph", slog.Any("err", err))
	} else if GraphvizSave.Value() {
		go func(g graph.Graph[int, *kilonova.SubTest]) {
			f, err := os.CreateTemp("", fmt.Sprintf("submission-graph-%d-*.gv", sub.ID))
			if err != nil {
				slog.WarnContext(ctx, "Couldn't save graph file", slog.Any("err", err))
				return
			}
			defer func() {
				if err := f.Close(); err != nil {
					slog.WarnContext(ctx, "Could not close graph file", slog.Any("err", err))
				}
			}()
			if err := draw.DOT(g, f); err != nil {
				slog.WarnContext(ctx, "Couldn't write graph file", slog.Any("err", err))
				return
			}
		}(g)
	}

	// TODO: This is shit.
	// It is basically 2 implementations for ~ the same thing. It could be merged neater
	switch sub.SubmissionType {
	case kilonova.EvalTypeClassic:
		if err := sh.handleClassicSubmission(ctx, checker, subTests); err != nil {
			slog.WarnContext(ctx, "Couldn't deal with classic submission", slog.Any("err", err))
			return err
		}
	case kilonova.EvalTypeICPC:
		if err := sh.handleICPCSubmission(ctx, checker, subTests); err != nil {
			slog.WarnContext(ctx, "Couldn't deal with ICPC submission", slog.Any("err", err))
			return err
		}
	default:
		return kilonova.Statusf(500, "Invalid eval type")
	}

	if err := datastore.GetBucket(datastore.BucketTypeCompiles).RemoveFile(fmt.Sprintf("%d.bin", sub.ID)); err != nil {
		slog.WarnContext(ctx, "Couldn't remove compilation artifact", slog.Any("err", err))
	}

	if err := checker.Cleanup(ctx); err != nil {
		slog.WarnContext(ctx, "Couldn't remove checker artifact", slog.Any("err", err))
	}
	return nil
}

func (sh *submissionHandler) handleClassicSubmission(ctx context.Context, checker checkers.Checker, subTests []*kilonova.SubTest) error {
	var wg sync.WaitGroup

	for _, subTest := range subTests {
		subTest := subTest
		wg.Add(1)

		go func() {
			defer wg.Done()
			_, _, err := sh.handleSubTest(ctx, checker, subTest)
			if err != nil {
				slog.WarnContext(ctx, "Error handling subtest", slog.Any("err", err))
			}
		}()
	}

	wg.Wait()

	if err := sh.scoreTests(ctx); err != nil {
		slog.WarnContext(ctx, "Couldn't score test", slog.Any("err", err))
	}

	return nil
}

func (sh *submissionHandler) handleICPCSubmission(ctx context.Context, checker checkers.Checker, subTests []*kilonova.SubTest) error {
	var failed bool
	var upd kilonova.SubmissionUpdate
	upd.Status = kilonova.StatusFinished

	for _, subTest := range subTests {
		if failed {
			if err := sh.base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{
				Done: &True, Skipped: &True,
				Verdict: &skippedVerdict,
			}); err != nil {
				slog.WarnContext(ctx, "Couldn't update skipped subtest", slog.Any("err", err))
			}
			continue
		}
		score, verdict, err := sh.handleSubTest(ctx, checker, subTest)
		if err != nil {
			slog.WarnContext(ctx, "Error handling subtest", slog.Any("err", err))
			continue
		}
		if !score.Equal(decimal.NewFromInt(100)) {
			upd.Score = &sh.pb.DefaultPoints
			upd.ChangeVerdict = true

			verdict = fmt.Sprintf("%s (test_verdict.test_x #%d)", strings.ReplaceAll(verdict, "translate:", "test_verdict."), subTest.VisibleID)
			upd.ICPCVerdict = &verdict

			failed = true
		}
	}

	if !failed {
		hundred := decimal.NewFromInt(100)
		upd.Score = &hundred
		upd.ChangeVerdict = true
		upd.ICPCVerdict = &acceptedVerdict
	}

	subTests, err := sh.base.SubTests(ctx, sh.sub.ID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get subtests for max score/mem updating", slog.Any("err", err))
		return err
	}

	var memory int
	var time float64
	for _, subtest := range subTests {
		memory = max(memory, subtest.Memory)
		time = max(time, subtest.Time)
	}
	upd.MaxTime = &time
	upd.MaxMemory = &memory

	return sh.base.UpdateSubmission(ctx, sh.sub.ID, upd)
}

func (sh *submissionHandler) compileSubmission(ctx context.Context) error {
	req, err := sh.genSubCompileRequest(ctx)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't generate compilation request", slog.Any("err", err))
		return fmt.Errorf("Couldn't generate compilation request: %w", err)
	}

	resp, err := tasks.CompileTask(ctx, sh.runner, req, graderLogger)
	if err != nil {
		return fmt.Errorf("Error from eval: %w", err)
	}

	var compileTime *float64
	if resp.Stats != nil {
		compileTime = &resp.Stats.Time
	}
	compileError := !resp.Success
	if err := sh.base.UpdateSubmission(ctx, sh.sub.ID, kilonova.SubmissionUpdate{
		CompileError: &compileError, CompileMessage: &resp.Output, CompileTime: compileTime,
	}); err != nil {
		spew.Dump(err)
		return fmt.Errorf("Couldn't update submission: %w", err)
	}

	if !resp.Success {
		compileErrVerdict := "test_verdict.compile_error"
		if err := sh.base.UpdateSubmission(ctx, sh.sub.ID, kilonova.SubmissionUpdate{
			Status: kilonova.StatusFinished, Score: &sh.pb.DefaultPoints,
			ChangeVerdict: true, ICPCVerdict: &compileErrVerdict,
		}); err != nil {
			return fmt.Errorf("Couldn't finalize submission with compiler error: %w", err)
		}
		stks, err := sh.base.SubmissionSubTasks(ctx, sh.sub.ID)
		if err != nil {
			return fmt.Errorf("Couldn't get submission subtasks: %w", err)
		}
		for _, stk := range stks {
			if err := sh.base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, decimal.Zero); err != nil {
				return fmt.Errorf("Couldn't finish subtasks: %w", err)
			}
		}
		return kilonova.Statusf(204, "Compile failed")
	}
	return nil
}

func (sh *submissionHandler) handleSubTest(ctx context.Context, checker checkers.Checker, subTest *kilonova.SubTest) (decimal.Decimal, string, error) {
	if subTest.TestID == nil {
		slog.ErrorContext(ctx, "A subtest whose test was purged was detected.", slog.Any("subtest", subTest))
		return decimal.Zero, "", kilonova.Statusf(400, "Trying to handle subtest whose test was purged. This should never happen")
	}

	execRequest := &tasks.ExecRequest{
		SubID:       sh.sub.ID,
		SubtestID:   subTest.ID,
		Filename:    sh.pb.TestName,
		MemoryLimit: sh.pb.MemoryLimit,
		TimeLimit:   sh.pb.TimeLimit,
		Lang:        sh.lang,
		TestID:      *subTest.TestID,
	}
	if sh.pb.ConsoleInput {
		execRequest.Filename = "stdin"
	}

	resp, err := tasks.ExecuteTask(ctx, sh.runner, int64(sh.pb.MemoryLimit), execRequest, graderLogger)
	if err != nil {
		return decimal.Zero, "", fmt.Errorf("Couldn't execute subtest: %w", err)
	}
	var testScore decimal.Decimal

	// Make sure TLEs are fully handled
	if resp.Time > sh.pb.TimeLimit {
		resp.Time = sh.pb.TimeLimit
		resp.Comments = "translate:timeout"
	}

	if resp.Comments == "" {
		resp.Comments, testScore = checker.RunChecker(ctx, subTest.ID, *subTest.TestID)
	}

	// Hide fatal signals for ICPC submissions
	if sh.sub.SubmissionType == kilonova.EvalTypeICPC {
		if strings.Contains(resp.Comments, "signal 9") {
			resp.Comments = "translate:memory_limit"
		}
		if strings.Contains(resp.Comments, "Caught fatal signal") || strings.Contains(resp.Comments, "Exited with error status") {
			resp.Comments = "translate:runtime_error"
		}
	}

	if err := sh.base.UpdateSubTest(ctx, subTest.ID, kilonova.SubTestUpdate{Memory: &resp.Memory, Percentage: &testScore, Time: &resp.Time, Verdict: &resp.Comments, Done: &True}); err != nil {
		return decimal.Zero, "", fmt.Errorf("Error during evaltest updating: %w", err)
	}
	return testScore, resp.Comments, nil
}

func (sh *submissionHandler) markSubtestsDone(ctx context.Context) error {
	sts, err := sh.base.SubTests(ctx, sh.sub.ID)
	if err != nil {
		return fmt.Errorf("Error during getting subtests: %w", err)
	}
	for _, st := range sts {
		if st.Done {
			continue
		}
		if err := sh.base.UpdateSubTest(ctx, st.ID, kilonova.SubTestUpdate{Done: &True}); err != nil {
			slog.WarnContext(ctx, "Couldn't mark subtest done", slog.Any("subtest", st), slog.Any("err", err))
		}
	}
	return nil
}

func (sh *submissionHandler) scoreTests(ctx context.Context) error {
	subtests, err1 := sh.base.SubTests(ctx, sh.sub.ID)
	if err1 != nil {
		return err1
	}

	subTasks, err1 := sh.base.SubmissionSubTasks(ctx, sh.sub.ID)
	if err1 != nil {
		return err1
	}
	if len(subTasks) == 0 {
		subTasks = nil
	}

	var score = sh.pb.DefaultPoints

	if len(subTasks) > 0 {
		subMap := make(map[int]*kilonova.SubTest)
		for _, st := range subtests {
			subMap[st.ID] = st
		}
		for _, stk := range subTasks {
			percentage := decimal.NewFromInt(100)
			if len(stk.Subtests) == 0 { // Empty subtasks should be invalidated
				percentage = decimal.Zero
			}
			for _, id := range stk.Subtests {
				st, ok := subMap[id]
				if !ok {
					slog.WarnContext(ctx, "Couldn't find subtest. This should not really happen.")
					continue
				}
				percentage = decimal.Min(percentage, st.Percentage)
			}
			// subTaskScore = stk.Score * (percentage / 100) rounded to the precision
			subTaskScore := stk.Score.Mul(percentage.Shift(-2)).Round(sh.pb.ScorePrecision)
			score = score.Add(subTaskScore)
			if err := sh.base.UpdateSubmissionSubtaskPercentage(ctx, stk.ID, percentage); err != nil {
				slog.WarnContext(ctx, "Couldn't update subtask percentage", slog.Any("err", err))
			}
		}
	} else {
		for _, subtest := range subtests {
			// testScore = subtest.Score * (subtest.Percentage / 100) rounded to the precision
			testScore := subtest.Score.Mul(subtest.Percentage.Shift(-2)).Round(sh.pb.ScorePrecision)
			score = score.Add(testScore)
		}
	}

	var memory int
	var time float64
	for _, subtest := range subtests {
		memory = max(memory, subtest.Memory)
		time = max(time, subtest.Time)
	}

	return sh.base.UpdateSubmission(ctx, sh.sub.ID, kilonova.SubmissionUpdate{Status: kilonova.StatusFinished, Score: &score, MaxTime: &time, MaxMemory: &memory})
}

var ForceSecureSandbox = config.GenFlag[bool]("feature.grader.force_secure_sandbox", true, "Force use of secure sandbox only. Should be always enabled in production environments")

func getAppropriateRunner(ctx context.Context) (eval.BoxScheduler, error) {
	var boxFunc scheduler.BoxFunc
	var boxVersion = "NONE"
	if scheduler.CheckCanRun(ctx, box.New) {
		boxFunc = box.New
		boxVersion = box.IsolateVersion()
	} else if scheduler.CheckCanRun(ctx, box.NewStupid) && !ForceSecureSandbox.Value() {
		slog.WarnContext(ctx, "Secure sandbox not found. Using stupid sandbox")
		boxFunc = box.NewStupid
		boxVersion = "stupid"
	}
	if boxFunc == nil {
		slog.ErrorContext(ctx, "Remote grader has not been implemented. No grader available!")
		os.Exit(1)
	}

	slog.InfoContext(ctx, "Trying to spin up local grader")
	bm, err := scheduler.New(config.Eval.StartingBox, config.Eval.NumConcurrent, config.Eval.GlobalMaxMem, graderLogger, boxFunc)
	if err != nil {
		return nil, err
	}
	slog.InfoContext(ctx, "Running local grader", slog.String("version", boxVersion))

	return bm, nil
}

func (sh *submissionHandler) getAppropriateChecker(ctx context.Context) (checkers.Checker, error) {
	if sh.settings.CheckerName == "" {
		return &checkers.DiffChecker{}, nil
	}
	att, err := sh.base.ProblemAttByName(ctx, sh.pb.ID, sh.settings.CheckerName)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get problem checker metadata: %w", err)
	}
	data, err := sh.base.ProblemAttDataByName(ctx, sh.pb.ID, sh.settings.CheckerName)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get problem checker code: %w", err)
	}
	subCode, err := sh.base.RawSubmissionCode(ctx, sh.sub.ID)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get submission source code: %w", err)
	}
	if sh.settings.LegacyChecker {
		return checkers.NewLegacyCustomChecker(sh.runner, graderLogger, sh.pb, sh.settings.CheckerName, data, subCode, att.LastUpdatedAt), nil
	}
	return checkers.NewStandardCustomChecker(sh.runner, graderLogger, sh.pb, sh.settings.CheckerName, data, subCode, att.LastUpdatedAt), nil
}
