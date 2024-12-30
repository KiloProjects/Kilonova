package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"text/template"
	"time"

	_ "embed"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/integrations/llm"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/shopspring/decimal"
)

func (s *API) maxScore(w http.ResponseWriter, r *http.Request) {
	var args struct {
		UserID int
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID <= 0 {
		if util.UserBrief(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.UserBrief(r).ID
	}

	returnData(w, s.base.MaxScore(r.Context(), args.UserID, util.Problem(r).ID))
}

func (s *API) problemStatistics(w http.ResponseWriter, r *http.Request) {
	stats, err := s.base.ProblemStatistics(r.Context(), util.Problem(r), util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, stats)
}

type scoreBreakdownRet struct {
	MaxScore decimal.Decimal               `json:"max_score"`
	Problem  *kilonova.Problem             `json:"problem"`
	Subtasks []*kilonova.SubmissionSubTask `json:"subtasks"`

	// ProblemEditor is true only if the request author is public
	// It does not take into consideration if the supplied user is the problem editor
	ProblemEditor bool `json:"problem_editor"`
	// Subtests are arranged from submission subtasks so something legible can be rebuilt to show more information on the subtasks
	Subtests []*kilonova.SubTest `json:"subtests"`
}

func (s *API) maxScoreBreakdown(w http.ResponseWriter, r *http.Request) {
	var args struct {
		UserID int

		ContestID  *int
		ViewFrozen bool `json:"view_frozen"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	var contest *kilonova.Contest
	if args.ContestID != nil {
		c, err := s.base.Contest(r.Context(), *args.ContestID)
		if err != nil {
			statusError(w, err)
			return
		}
		contest = c
	}

	// This endpoint may leak stuff that shouldn't be generally seen (like in contests), so restrict this option to editors only
	// OR to contest editors/testers when ContestID is supplied
	// It isn't used anywhere right now, but it might be useful in the future
	if !(s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)) || (contest != nil && contest.IsTester(util.UserBrief(r)))) {
		args.UserID = -1
	}
	if args.UserID <= 0 {
		if util.UserBrief(r) == nil {
			errorData(w, "No user specified", 400)
			return
		}
		args.UserID = util.UserBrief(r).ID
	}

	var maxScore decimal.Decimal
	if contest == nil {
		maxScore = s.base.MaxScore(r.Context(), args.UserID, util.Problem(r).ID)
	} else {
		maxScore = s.base.ContestMaxScore(
			r.Context(), args.UserID, util.Problem(r).ID, contest.ID,
			s.base.UserContestFreezeTime(util.UserBrief(r), contest, args.ViewFrozen),
		)
	}

	switch util.Problem(r).ScoringStrategy {
	case kilonova.ScoringTypeMaxSub, kilonova.ScoringTypeICPC:
		id, err := s.base.MaxScoreSubID(r.Context(), args.UserID, util.Problem(r).ID)
		if err != nil {
			statusError(w, err)
			return
		}
		if id <= 0 {
			returnData(w, scoreBreakdownRet{
				MaxScore: maxScore,
				Problem:  util.Problem(r),
				Subtasks: []*kilonova.SubmissionSubTask{},
				Subtests: []*kilonova.SubTest{},

				ProblemEditor: s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)),
			})
			return
		}
		sub, err := s.base.Submission(r.Context(), id, util.UserBrief(r))
		if err != nil {
			statusError(w, err)
			return
		}

		returnData(w, scoreBreakdownRet{
			MaxScore: maxScore,
			Problem:  util.Problem(r),
			Subtasks: sub.SubTasks,
			Subtests: sub.SubTests,

			ProblemEditor: s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)),
		})
	case kilonova.ScoringTypeSumSubtasks:
		stks, err := s.base.MaximumScoreSubTasks(r.Context(), util.Problem(r).ID, args.UserID, args.ContestID)
		if err != nil {
			statusError(w, err)
			return
		}

		tests, err := s.base.MaximumScoreSubTaskTests(r.Context(), util.Problem(r).ID, args.UserID, args.ContestID)
		if err != nil {
			statusError(w, err)
			return
		}

		returnData(w, scoreBreakdownRet{
			MaxScore: maxScore,
			Problem:  util.Problem(r),
			Subtasks: stks,
			Subtests: tests,

			ProblemEditor: s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r)),
		})
	default:
		slog.WarnContext(r.Context(), "Unknown problem scoring type", slog.Any("type", util.Problem(r).ScoringStrategy))
		errorData(w, "Unknown problem scoring type", 500)
		return
	}

}

func (s *API) deleteProblem(w http.ResponseWriter, r *http.Request) {
	if err := s.base.DeleteProblem(context.WithoutCancel(r.Context()), util.Problem(r)); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, "Deleted problem")
}

var (
	//go:embed templData/default_en_statement.md
	enPbStatementStr string
	//go:embed templData/default_ro_statement.md
	roPbStatementStr string

	defaultEnProblemStatement = template.Must(template.New("enStmt").Parse(enPbStatementStr))
	defaultRoProblemStatement = template.Must(template.New("enStmt").Parse(roPbStatementStr))
)

func (s *API) addStubStatement(ctx context.Context, pb *kilonova.Problem, lang *string, author *kilonova.UserBrief) error {
	if lang == nil || *lang == "" {
		return nil
	}

	if !(*lang == "en" || *lang == "ro") {
		return kilonova.Statusf(400, "Invalid statement language")
	}

	var attTempl *template.Template
	if *lang == "en" {
		attTempl = defaultEnProblemStatement
	} else if *lang == "ro" {
		attTempl = defaultRoProblemStatement
	} else {
		slog.WarnContext(ctx, "Unknown language", slog.String("lang", *lang))
		return nil
	}

	inFile := "stdin"
	outFile := "stdout"
	if !pb.ConsoleInput {
		inFile = pb.TestName + ".in"
		outFile = pb.TestName + ".out"
	}
	var buf bytes.Buffer
	if err := attTempl.Execute(&buf, struct {
		InputFile  string
		OutputFile string
	}{InputFile: inFile, OutputFile: outFile}); err != nil {
		slog.WarnContext(ctx, "Template rendering error", slog.Any("err", err))
		return nil
	}
	if err := s.base.CreateProblemAttachment(ctx, &kilonova.Attachment{
		Visible: false,
		Private: false,
		Exec:    false,
		Name:    fmt.Sprintf("statement-%s.md", *lang),
	}, pb.ID, &buf, &author.ID); err != nil {
		slog.WarnContext(ctx, "Could not create problem attachment", slog.Any("err", err))
	}
	return nil
}

func (s *API) initProblem(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Title        string `json:"title"`
		ConsoleInput bool   `json:"consoleInput"`

		StatementLang *string `json:"statementLang"`
		ProblemListID *int    `json:"pblistID"`
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}

	// Do the check before problem creation because it'd be awkward to create the problem and then show the error
	if args.StatementLang != nil && !(*args.StatementLang == "" || *args.StatementLang == "en" || *args.StatementLang == "ro") {
		errorData(w, "Invalid statement language", 400)
		return
	}

	pb, err := s.base.CreateProblem(r.Context(), args.Title, util.UserBrief(r), args.ConsoleInput)
	if err != nil {
		statusError(w, err)
		return
	}

	if args.ProblemListID != nil {
		list, err := s.base.ProblemList(r.Context(), *args.ProblemListID)
		if err == nil {
			list.List = append(list.List, pb.ID)
			if err := s.base.UpdateProblemListProblems(r.Context(), list.ID, list.List); err != nil {
				slog.WarnContext(r.Context(), "Could not update list problems", slog.Any("err", err))
			}
		}
	}

	if err := s.addStubStatement(r.Context(), pb, args.StatementLang, util.UserBrief(r)); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, pb.ID)
}

func (s *API) importProblemArchive(w http.ResponseWriter, r *http.Request) {
	pb, err := s.base.CreateProblem(r.Context(), "unnamed", util.UserBrief(r), true)
	if err != nil {
		statusError(w, err)
		return
	}

	r = r.WithContext(context.WithValue(r.Context(), util.ProblemKey, pb))
	if err := s.processArchive(r, true); err != nil {
		statusError(w, err)
		return
	}

	// Get problem after most likely setting new properties after import
	if pb2, err := s.base.Problem(r.Context(), pb.ID); err != nil {
		slog.ErrorContext(r.Context(), "Could not get problem again", slog.Any("err", err))
	} else {
		pb = pb2
	}

	var args struct {
		StatementLang *string `json:"statementLang"`
		ProblemListID *int    `json:"pblistID"`
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if args.ProblemListID != nil {
		list, err := s.base.ProblemList(r.Context(), *args.ProblemListID)
		if err == nil {
			list.List = append(list.List, pb.ID)
			if err := s.base.UpdateProblemListProblems(r.Context(), list.ID, list.List); err != nil {
				slog.WarnContext(r.Context(), "Could not update problem list problems", slog.Any("err", err))
			}
		}
	}

	if err := s.addStubStatement(r.Context(), pb, args.StatementLang, util.UserBrief(r)); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, pb.ID)
}

func (s *API) getProblems(ctx context.Context, args kilonova.ProblemFilter) ([]*kilonova.Problem, error) {
	args.Look = true
	args.LookingUser = util.UserBriefContext(ctx)

	return s.base.Problems(ctx, args)
}

type problemSearchResult struct {
	Problems []*sudoapi.FullProblem `json:"problems"`

	Count int `json:"count"`
}

func (s *API) searchProblems(ctx context.Context, args kilonova.ProblemFilter) (*problemSearchResult, error) {
	args.Look = true
	args.LookingUser = util.UserBriefContext(ctx)

	if args.Limit == 0 || args.Limit > 50 {
		args.Limit = 50
	}

	var scoreUser = util.UserBriefContext(ctx)
	if args.ScoreUserID != nil {
		user, err := s.base.UserBrief(ctx, *args.ScoreUserID)
		if err != nil {
			return nil, err
		}
		scoreUser = user
	}

	problems, cnt, err := s.base.SearchProblems(ctx, args, scoreUser, util.UserBriefContext(ctx))
	if err != nil {
		return nil, err
	}
	return &problemSearchResult{Problems: problems, Count: cnt}, nil
}

func (s *API) updateProblem(ctx context.Context, args kilonova.ProblemUpdate) error {
	return s.base.UpdateProblem(ctx, util.ProblemContext(ctx).ID, args, util.UserBriefContext(ctx))
}

func (s *API) translateProblemStatement() http.HandlerFunc {
	var translateMu sync.Mutex
	return func(w http.ResponseWriter, r *http.Request) {
		var args struct {
			Model string `json:"model"`
		}
		if err := parseRequest(r, &args); err != nil {
			statusError(w, err)
			return
		}
		if !translateMu.TryLock() {
			errorData(w, "Will not process more than one pending translation at once. Please try again later.", 400)
			return
		}
		defer translateMu.Unlock()
		att, err := s.base.ProblemAttByName(r.Context(), util.Problem(r).ID, "statement-ro.md")
		if err != nil {
			statusError(w, err)
			return
		}
		data, err := s.base.AttachmentData(r.Context(), att.ID)
		if err != nil {
			statusError(w, err)
			return
		}
		t := time.Now()
		output, err := llm.TranslateStatement(r.Context(), string(data), args.Model)
		if err != nil {
			errorData(w, err, 400)
			return
		}
		s.base.LogUserAction(r.Context(), "Triggered LLM translation", slog.String("model", args.Model), slog.Any("problem", util.Problem(r)), slog.Duration("duration", time.Since(t)))
		att2, err := s.base.ProblemAttByName(r.Context(), util.Problem(r).ID, "statement-en-llm.md")
		if err != nil {
			if errors.Is(err, kilonova.ErrNotFound) {
				att2 = &kilonova.Attachment{Name: "statement-en-llm.md"}
				err = s.base.CreateProblemAttachment(r.Context(), att2, util.Problem(r).ID, strings.NewReader(output), &util.UserBrief(r).ID)
				if err != nil {
					statusError(w, err)
				}
				returnData(w, "Created translation")
				return
			}
			statusError(w, err)
			return
		}
		if err := s.base.UpdateAttachmentData(r.Context(), att2.ID, []byte(output), util.UserBrief(r)); err != nil {
			statusError(w, err)
			return
		}

		returnData(w, "Updated translation")
	}
}

func boolPtrString(val *bool) string {
	if val == nil {
		return "N/A"
	}
	if *val {
		return "true"
	}
	return "false"
}

func (s *API) togglePblistProblems(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Deep         bool  `json:"deep"`
		Visible      *bool `json:"visible"`
		VisibleTests *bool `json:"visibleTests"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if err := s.base.ToggleDeepPbListProblems(r.Context(), util.ProblemList(r), args.Deep, kilonova.ProblemUpdate{Visible: args.Visible, VisibleTests: args.VisibleTests}); err != nil {
		statusError(w, err)
		return
	}

	s.base.LogUserAction(r.Context(), "Bulk updated problem lists",
		slog.Any("problem_list", util.ProblemList(r)),
		slog.String("visible_problems", boolPtrString(args.Visible)), slog.String("downloadable_tests", boolPtrString(args.VisibleTests)),
		slog.Bool("deep", args.Deep),
	)

	returnData(w, "Updated problems")
}

func (s *API) addProblemEditor(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.AddProblemEditor(r.Context(), util.Problem(r).ID, user.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Added problem editor")
}

func (s *API) addProblemViewer(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if user.ID == util.UserBrief(r).ID {
		errorData(w, "You can't demote yourself to viewer rank!", 400)
		return
	}

	if err := s.base.AddProblemViewer(r.Context(), util.Problem(r).ID, user.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Added problem viewer")
}

func (s *API) stripProblemAccess(ctx context.Context, args struct {
	UserID int `json:"user_id"`
}) error {
	if args.UserID == util.UserBriefContext(ctx).ID {
		return kilonova.Statusf(400, "You can't strip your own access!")
	}

	return s.base.StripProblemAccess(ctx, util.ProblemContext(ctx).ID, args.UserID)
}

type problemAccessControl struct {
	Editors []*kilonova.UserBrief `json:"editors"`
	Viewers []*kilonova.UserBrief `json:"viewers"`
}

func (s *API) getProblemAccessControl(ctx context.Context, _ struct{}) (*problemAccessControl, error) {
	editors, err := s.base.ProblemEditors(ctx, util.ProblemContext(ctx).ID)
	if err != nil {
		return nil, err
	}

	viewers, err := s.base.ProblemViewers(ctx, util.ProblemContext(ctx).ID)
	if err != nil {
		return nil, err
	}

	return &problemAccessControl{
		Editors: editors,
		Viewers: viewers,
	}, nil
}

func (s *API) problemLanguages(ctx context.Context, _ struct{}) ([]*sudoapi.Language, error) {
	return s.base.ProblemLanguages(ctx, util.ProblemContext(ctx).ID)
}

func (s *API) getProblem(ctx context.Context, _ struct{}) (*kilonova.Problem, error) {
	return util.ProblemContext(ctx), nil
}
