package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/archive/test"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

type Assets struct {
	base *sudoapi.BaseAPI
}

func NewAssets(base *sudoapi.BaseAPI) *Assets {
	return &Assets{base}
}

func (s *Assets) initSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, err := s.base.SessionUser(r.Context(), s.base.GetSessCookie(r))
		if err != nil || user == nil {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), util.UserKey, user)))
	})
}

func (s *Assets) AssetsRouter() http.Handler {
	r := chi.NewRouter()
	api := New(s.base)

	r.Use(s.initSession)

	r.Route("/problem/{problemID}", func(r chi.Router) {
		r.Use(api.validateProblemID)
		r.Use(api.validateProblemVisible)

		r.With(api.validateVisibleTests, api.validateTestID).Get("/test/{tID}/input", s.ServeTestInput)
		r.With(api.validateVisibleTests, api.validateTestID).Get("/test/{tID}/output", s.ServeTestOutput)

		r.With(api.validateProblemFullyVisible).Get("/problemArchive", s.ServeProblemArchive)

		r.With(api.validateAttachmentName).Get("/attachment/{aName}", s.ServeAttachment)
		r.With(api.validateAttachmentID).Get("/attachmentByID/{aID}", s.ServeAttachment)
	})

	r.Route("/blogPost/{bpName}", func(r chi.Router) {
		r.Use(api.validateBlogPostName)
		r.Use(api.validateBlogPostVisible)

		r.With(api.validateAttachmentName).Get("/attachment/{aName}", s.ServeAttachment)
		r.With(api.validateAttachmentID).Get("/attachmentByID/{aID}", s.ServeAttachment)
	})

	r.With(api.MustBeProposer).Get("/subtest/{subtestID}", s.ServeSubtest)

	r.With(api.validateContestID, api.validateContestEditor).Get("/contest/{contestID}/leaderboard.csv", s.ServeContestLeaderboard)

	return r
}

func (s *Assets) ServeAttachment(w http.ResponseWriter, r *http.Request) {
	w.Header().Add("X-Robots-Tag", "noindex, nofollow, noarchive")
	att := util.Attachment(r)

	attData, err := s.base.AttachmentData(r.Context(), att.ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get attachment data", 500)
		return
	}

	w.Header().Set("Cache-Control", `public, max-age=3600`)

	// If markdown file and client asks for HTML format, render the markdown
	if path.Ext(att.Name) == ".md" && r.FormValue("format") == "html" {
		data, err := s.base.RenderMarkdown(attData, &kilonova.RenderContext{Problem: util.Problem(r), BlogPost: util.BlogPost(r)})
		if err != nil {
			zap.S().Warn(err)
			http.Error(w, "Could not render file", 500)
			return
		}
		http.ServeContent(w, r, att.Name+".html", att.LastUpdatedAt, bytes.NewReader(data))
		return
	}

	http.ServeContent(w, r, att.Name, att.LastUpdatedAt, bytes.NewReader(attData))
}

func (s *Assets) ServeContestLeaderboard(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Frozen bool `json:"frozen"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		http.Error(w, "Can't decode parameters", 400)
		return
	}

	ld, err := s.base.ContestLeaderboard(
		r.Context(), util.Contest(r),
		s.base.UserContestFreezeTime(util.UserBrief(r), util.Contest(r), args.Frozen),
	)
	if err != nil {
		http.Error(w, err.Error(), err.Code)
		return
	}
	var buf bytes.Buffer
	wr := csv.NewWriter(&buf)

	// Header
	header := []string{"username"}
	for _, pb := range ld.ProblemOrder {
		name, ok := ld.ProblemNames[pb]
		if !ok {
			zap.S().Warn("Invalid s.base.ContestLeaderboard output")
			http.Error(w, "Invalid internal data", 500)
			continue
		}
		header = append(header, name)
	}
	if util.Contest(r).LeaderboardStyle == kilonova.LeaderboardTypeICPC {
		header = append(header, "num_solved")
		header = append(header, "penalty")
	} else {
		header = append(header, "total")
	}
	if err := wr.Write(header); err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't write CSV", 500)
		return
	}
	for _, entry := range ld.Entries {
		line := []string{entry.User.Name}
		for _, pb := range ld.ProblemOrder {
			score, ok := entry.ProblemScores[pb]
			if !ok {
				line = append(line, "-")
			} else {
				line = append(line, score.String())
			}
		}

		line = append(line, entry.TotalScore.String())
		if err := wr.Write(line); err != nil {
			zap.S().Warn(err)
			http.Error(w, "Couldn't write CSV", 500)
			return
		}
	}

	wr.Flush()
	if err := wr.Error(); err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't write CSV", 500)
		return
	}

	http.ServeContent(w, r, "leaderboard.csv", time.Now(), bytes.NewReader(buf.Bytes()))
}

func (s *Assets) ServeSubtest(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(chi.URLParam(r, "subtestID"))
	if err != nil {
		http.Error(w, "Bad ID", 400)
		return
	}
	subtest, err1 := s.base.SubTest(r.Context(), id)
	if err1 != nil {
		http.Error(w, "Invalid subtest", 400)
		return
	}
	sub, err1 := s.base.Submission(r.Context(), subtest.SubmissionID, util.UserBrief(r))
	if err1 != nil {
		zap.S().Warn(err1)
		http.Error(w, "You aren't allowed to do that", 500)
		return
	}

	if !s.base.IsProblemEditor(util.UserBrief(r), sub.Problem) {
		http.Error(w, "You aren't allowed to do that!", http.StatusUnauthorized)
		return
	}

	rc, err := s.base.SubtestReader(subtest.ID)
	if err != nil {
		http.Error(w, "The subtest may have been purged as a routine data-saving process", 404)
		return
	}
	defer rc.Close()
	http.ServeContent(w, r, "subtest.out", time.Now(), rc)
}

func (s *Assets) ServeTestInput(w http.ResponseWriter, r *http.Request) {
	rr, err := s.base.TestInput(util.Test(r).ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get test input", 500)
		return
	}
	defer rr.Close()

	tname := fmt.Sprintf("%d-%s.in", util.Test(r).VisibleID, util.Problem(r).TestName)

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", tname))
	http.ServeContent(w, r, tname, time.Unix(0, 0), rr)
}

func (s *Assets) ServeTestOutput(w http.ResponseWriter, r *http.Request) {
	rr, err := s.base.TestOutput(util.Test(r).ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get test output", 500)
		return
	}
	defer rr.Close()

	tname := fmt.Sprintf("%d-%s.out", util.Test(r).VisibleID, util.Problem(r).TestName)

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", tname))
	http.ServeContent(w, r, tname, time.Unix(0, 0), rr)
}

func (s *Assets) ServeProblemArchive(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args test.ArchiveGenOptions
	if err := decoder.Decode(&args, r.Form); err != nil {
		http.Error(w, "Can't decode parameters", 400)
		return
	}

	args.Tests = args.Tests && s.base.CanViewTests(util.UserBrief(r), util.Problem(r))
	args.PrivateAttachments = args.PrivateAttachments && s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r))
	args.AllSubmissions = args.AllSubmissions && s.base.IsProblemEditor(util.UserBrief(r), util.Problem(r))
	args.SubsLook = true
	args.SubsLookingUser = util.UserBrief(r)

	w.Header().Add("Content-Type", "application/zip")
	w.Header().Add("Content-Disposition", fmt.Sprintf(`attachment; filename="%d-%s.zip"`, util.Problem(r).ID, kilonova.MakeSlug(util.Problem(r).Name)))
	w.WriteHeader(200)

	wr := bufio.NewWriter(w)
	if err := test.GenerateArchive(r.Context(), util.Problem(r), wr, s.base, &args); err != nil {
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
		fmt.Fprint(w, err)
	}
	if err := wr.Flush(); err != nil {
		zap.S().Warn(err)
	}
}
