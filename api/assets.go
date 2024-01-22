package api

import (
	"bufio"
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"image"
	"io"
	"math"
	"net/http"
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/archive/test"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/Yiling-J/theine-go"
	"github.com/disintegration/gift"
	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"image/jpeg"
	"image/png"
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

		// Enforce authed user for rate limit
		r.With(api.MustBeAuthed, api.validateProblemFullyVisible).Get("/problemArchive", s.ServeProblemArchive())

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
		if !errors.Is(err, context.Canceled) {
			zap.S().Warn(err)
		}
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

	mimeType := http.DetectContentType(attData)
	if (mimeType == "image/png" || mimeType == "image/jpeg") && (r.FormValue("w") != "" || r.FormValue("h") != "") {
		var ok = true
		width, height := 0, 0
		if r.FormValue("w") != "" {
			width2, err := strconv.Atoi(r.FormValue("w"))
			if err != nil {
				ok = false
			}
			width = width2
		}
		if r.FormValue("h") != "" {
			height2, err := strconv.Atoi(r.FormValue("h"))
			if err != nil {
				ok = false
			}
			height = height2
		}
		if width > 6000 || height > 6000 {
			// if it's too big, it's just eating up resources
			ok = false
		}

		renderType := fmt.Sprintf("img_%dx%d", width, height)
		if s.base.HasAttachmentRender(att.ID, renderType) {
			data, err := s.base.GetAttachmentRender(att.ID, renderType)
			if err != nil {
				zap.S().Warn(err)
			} else {
				defer data.Close()
				cachedData, err := io.ReadAll(data)
				if err != nil {
					zap.S().Warn(err)
				}
				http.ServeContent(w, r, att.Name, att.LastUpdatedAt, bytes.NewReader(cachedData))
				return
			}
		}

		src, _, err := image.Decode(bytes.NewReader(attData))
		if err != nil {
			zap.S().Debug(err)
			ok = false
		}
		if ok {
			g := gift.New(gift.Resize(width, height, gift.LanczosResampling))
			dst := image.NewRGBA(g.Bounds(src.Bounds()))
			g.Draw(dst, src)
			var buf bytes.Buffer
			if mimeType == "image/png" {
				png.Encode(&buf, dst)
			} else if mimeType == "image/jpeg" {
				jpeg.Encode(&buf, dst, nil)
			} else {
				zap.S().Warn("We somehow got here")
			}
			// Also cache it if's relatively small
			if width <= 4000 && height <= 4000 {
				s.base.SaveAttachmentRender(att.ID, renderType, buf.Bytes())
			}
			http.ServeContent(w, r, att.Name, att.LastUpdatedAt, bytes.NewReader(buf.Bytes()))
			return
		}
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
		if util.Contest(r).LeaderboardStyle == kilonova.LeaderboardTypeICPC {
			for _, pb := range ld.ProblemOrder {
				score, ok := entry.ProblemScores[pb]
				if !ok || score.Equal(decimal.NewFromInt(-1)) {
					line = append(line, "-")
					continue
				}
				if score.LessThan(decimal.NewFromInt(100)) {
					if val, ok := entry.ProblemAttempts[pb]; ok {
						line = append(line, strconv.Itoa(-val))
					} else {
						line = append(line, "-")
					}
				} else {
					col := "+"
					if val, ok := entry.ProblemAttempts[pb]; ok && val > 1 {
						col += strconv.Itoa(val)
					}
					if dur, ok := entry.ProblemTimes[pb]; ok {
						dur = math.Floor(dur)
						h, m := int(dur)/60, int(dur)%60
						col += fmt.Sprintf(" / %02d:%02d", h, m)
					}
					line = append(line, col)
				}
			}
			line = append(line, strconv.Itoa(entry.NumSolved), strconv.Itoa(entry.Penalty))
		} else {
			for _, pb := range ld.ProblemOrder {
				score, ok := entry.ProblemScores[pb]
				if !ok || score.Equal(decimal.NewFromInt(-1)) {
					line = append(line, "-")
				} else {
					line = append(line, score.String())
				}
			}
			line = append(line, entry.TotalScore.String())
		}

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
		http.Error(w, "Couldn't get submission", 500)
		return
	}

	if !s.base.IsProblemEditor(util.UserBrief(r), sub.Problem) {
		http.Error(w, "You aren't allowed to do that!", http.StatusUnauthorized)
		return
	}

	rc, err := s.base.GraderStore().SubtestReader(subtest.ID)
	if err != nil {
		http.Error(w, "The subtest may have been purged as a routine data-saving process", 404)
		return
	}
	defer rc.Close()
	w.WriteHeader(200)
	io.Copy(w, rc)
}

func (s *Assets) ServeTestInput(w http.ResponseWriter, r *http.Request) {
	rr, err := s.base.GraderStore().TestInput(util.Test(r).ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get test input", 500)
		return
	}
	defer rr.Close()

	tname := fmt.Sprintf("%d-%s.in", util.Test(r).VisibleID, util.Problem(r).TestName)

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", tname))
	w.WriteHeader(200)
	io.Copy(w, rr)
}

func (s *Assets) ServeTestOutput(w http.ResponseWriter, r *http.Request) {
	rr, err := s.base.GraderStore().TestOutput(util.Test(r).ID)
	if err != nil {
		zap.S().Warn(err)
		http.Error(w, "Couldn't get test output", 500)
		return
	}
	defer rr.Close()

	tname := fmt.Sprintf("%d-%s.out", util.Test(r).VisibleID, util.Problem(r).TestName)

	w.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", tname))
	w.WriteHeader(200)
	io.Copy(w, rr)
}

func (s *Assets) ServeProblemArchive() http.HandlerFunc {
	// If people try to download archives on 1000 different accounts at the same time i think we have a different problem
	pbArchiveUserCache, err := theine.NewBuilder[int, *sync.Mutex](1000).BuildWithLoader(func(ctx context.Context, key int) (theine.Loaded[*sync.Mutex], error) {
		var mu sync.Mutex
		return theine.Loaded[*sync.Mutex]{
			Value: &mu,
			Cost:  1,
			TTL:   1 * time.Hour,
		}, nil
	})
	if err != nil {
		zap.S().Fatal(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		mu, err := pbArchiveUserCache.Get(r.Context(), util.UserBrief(r).ID)
		if err != nil || mu == nil {
			zap.S().Warn(err)
			http.Error(w, "Could not aquire mutex", 500)
			return
		}
		if !mu.TryLock() {
			http.Error(w, "You cannot download more than one archive at once!", http.StatusForbidden)
			return
		}
		defer mu.Unlock()
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
				if !errors.Is(err, syscall.EPIPE) {
					zap.S().Warn(err)
				}
			}
			fmt.Fprint(w, err)
		}
		if err := wr.Flush(); err != nil && !errors.Is(err, syscall.EPIPE) {
			zap.S().Warn(err)
		}
	}
}
