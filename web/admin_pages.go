package web

import (
	"archive/zip"
	"bytes"
	"cmp"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/netip"
	"runtime/metrics"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/infra/maxmind"
	"github.com/KiloProjects/kilonova/internal/contestgen"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/adminviews"
	"github.com/KiloProjects/kilonova/web/ui/authviews"
	"github.com/gorilla/schema"
	g "maragu.dev/gomponents"
)

func (rt *Web) adminPanel(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title: kilonova.GetText(util.Language(r), "panel.admin"),
		Content: g.Group{adminviews.Tabs(r.Context(), "/admin"), adminviews.Panel(r.Context(), adminviews.PanelParams{
			AuthedUserID:  user.UserBrief(r).ID,
			BoolFlags:     config.GetFlags[bool](),
			StringFlags:   config.GetFlags[string](),
			IntFlags:      config.GetFlags[int](),
			IntArrayFlags: config.GetFlags[[]int](),
			StrArrayFlags: config.GetFlags[[]string](),
		})},
	})
}

func (rt *Web) userGen(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "user_gen.title"),
		Content: g.Group{adminviews.Tabs(r.Context(), "/admin/user_gen"), adminviews.UserGen(r.Context(), flags.DefaultLanguage(r.Context()))},
	})
}

func (rt *Web) contestGen(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "contest_gen.title"),
		Content: g.Group{adminviews.Tabs(r.Context(), "/admin/contest_gen"), adminviews.ContestGen(r.Context())},
	})
}

// contestGenRun creates the accounts in an uploaded profiles bundle and replies with a zip of the results and the audit log.
func (rt *Web) contestGenRun(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	f, _, err := r.FormFile("bundle")
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Missing profiles bundle")
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(f)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, "Could not read profiles bundle")
		return
	}
	cfg, err := contestgen.Parse(raw)
	if err != nil {
		rt.statusPage(w, r, http.StatusBadRequest, err.Error())
		return
	}

	var logBuf bytes.Buffer
	auditLog := slog.New(slog.NewJSONHandler(&logBuf, nil))
	// Don't stop halfway through if the admin closes the tab: created accounts would be left without their passwords.
	ctx := context.WithoutCancel(r.Context())
	slog.InfoContext(ctx, "Running contest account generation", slog.Any("user", user.UserBrief(r)), slog.Int("profiles", len(cfg.Profiles)))
	res, err := contestgen.Run(ctx, cfg, contestgen.LocalBackend(rt.base), auditLog, r.FormValue("dryRun") != "")
	if err != nil {
		rt.statusPage(w, r, kilonova.ErrorCode(err), err.Error())
		return
	}
	files, err := res.Files()
	if err != nil {
		rt.statusPage(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	files["generator.log"] = logBuf.Bytes()

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="contest-gen-`+time.Now().Format("2006-01-02T15-04-05")+`.zip"`)
	zw := zip.NewWriter(w)
	for name, data := range files {
		fw, err := zw.Create(name)
		if err == nil {
			_, err = fw.Write(data)
		}
		if err != nil {
			slog.ErrorContext(ctx, "Could not write contest generation archive", slog.Any("err", err))
			return
		}
	}
	if err := zw.Close(); err != nil {
		slog.ErrorContext(ctx, "Could not write contest generation archive", slog.Any("err", err))
	}
}

func (rt *Web) auditLog() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, err := strconv.Atoi(r.FormValue("page"))
		if err != nil {
			page = 0
		}

		logs, err := rt.base.GetAuditLogs(r.Context(), 50, (page-1)*50)
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch logs")
			return
		}

		numLogs, err := rt.base.GetLogCount(r.Context())
		if err != nil {
			rt.statusPage(w, r, 500, "Couldn't fetch log count")
			return
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "panel.audit_log"),
			Content: g.Group{adminviews.Tabs(r.Context(), "/admin/auditLog"), adminviews.AuditLog(r.Context(), adminviews.AuditLogParams{
				Logs:       logs,
				Pagination: pageNav(page, numLogs, 50),
			})},
		})
	}
}

// pageNav returns a paginator for count items split in pages of perPage, or nil if there's a single page.
func pageNav(page, count, perPage int) g.Node {
	numPages := (count + perPage - 1) / perPage
	if numPages <= 1 {
		return nil
	}
	return ui.Paginator(ui.PaginatorConfig{Page: page, NumPages: numPages, ShowArrows: true})
}

func (rt *Web) debugPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		all := metrics.All()
		var descs = make(map[string]string)
		samples := make([]metrics.Sample, 0, len(all))
		for _, m := range all {
			if m.Kind == metrics.KindFloat64 || m.Kind == metrics.KindUint64 {
				descs[m.Name] = m.Description
				samples = append(samples, metrics.Sample{Name: m.Name})
			}
		}
		metrics.Read(samples)
		finalMetrics := make([]*adminviews.Metric, 0, len(samples))
		for _, sample := range samples {
			desc, ok := descs[sample.Name]
			if !ok {
				slog.WarnContext(r.Context(), "Could not find description for metric", slog.String("name", sample.Name))
				desc = "No description provided"
			}
			m := &adminviews.Metric{
				Name:        sample.Name,
				Description: desc,
			}
			switch sample.Value.Kind() {
			case metrics.KindFloat64:
				m.Float = new(sample.Value.Float64())
			case metrics.KindUint64:
				m.Int = new(sample.Value.Uint64())
			default:
				slog.WarnContext(r.Context(), "Unknown metric type", slog.String("name", sample.Name), slog.Any("kind", sample.Value.Kind()))
			}
			finalMetrics = append(finalMetrics, m)
		}

		var stats = make([]*datastore.BucketStats, 0, 16)
		for _, bucket := range rt.base.DataStore().GetAll() {
			stats = append(stats, bucket.Statistics(false))
		}
		slices.SortFunc(stats, func(a, b *datastore.BucketStats) int { return cmp.Compare(a.Name, b.Name) })

		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "debug_info"),
			Content: g.Group{adminviews.Tabs(r.Context(), "/admin/debug"), adminviews.Debug(r.Context(), adminviews.DebugParams{Metrics: finalMetrics, BucketStats: stats})},
		})
	}
}

func (rt *Web) graderInfo() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		langs := rt.base.GraderLanguages(r.Context())
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "panel.grader"),
			Content: g.Group{g.Iff(user.UserBrief(r).IsAdmin(), func() g.Node { return adminviews.Tabs(r.Context(), "/grader") }), adminviews.GraderInfo(r.Context(), langs)},
		})
	}
}

// renderSessions loads per-session details (owner, devices, IP location) and renders the sessions page.
// contentUser is the user whose sessions are shown, or nil for the admin filter page.
func (rt *Web) renderSessions(w http.ResponseWriter, r *http.Request, contentUser *kilonova.UserFull, sessions []*sudoapi.Session, pagination g.Node) {
	isAdmin := user.UserBrief(r).IsAdmin()
	entries := make([]*authviews.SessionEntry, 0, len(sessions))
	for _, sess := range sessions {
		entry := &authviews.SessionEntry{Session: sess}
		if isAdmin {
			if contentUser == nil {
				if u, err := rt.base.UserBrief(r.Context(), sess.UserID); err == nil {
					entry.User = u
				}
			}
			devices, err := rt.base.SessionDevices(r.Context(), sess.ID)
			if err != nil {
				slog.WarnContext(r.Context(), "Could not get session devices", slog.Any("err", err))
			}
			for _, dev := range devices {
				de := &authviews.SessionDeviceEntry{SessionDevice: dev}
				if dev.IPAddr != nil {
					if data, err := maxmind.IPData(*dev.IPAddr); err == nil {
						de.Location = data
					}
				}
				entry.Devices = append(entry.Devices, de)
			}
		}
		entries = append(entries, entry)
	}

	title := kilonova.GetText(util.Language(r), "manage_sessions")
	var tabs g.Node // the admin bar only belongs on the /admin/sessions filter page
	if contentUser != nil {
		title += ": " + contentUser.Name
	} else {
		tabs = adminviews.Tabs(r.Context(), "/admin/sessions")
	}
	rt.runLayout(w, r, &LayoutParams{
		Title: title,
		Content: g.Group{tabs, authviews.Sessions(r.Context(), authviews.SessionsParams{
			IsAdmin:    isAdmin,
			Sessions:   entries,
			Pagination: pagination,
		})},
	})
}

func (rt *Web) userSessionsPage(w http.ResponseWriter, r *http.Request, user *kilonova.UserFull) {
	sessions, err := rt.base.UserSessions(r.Context(), user.ID)
	if err != nil {
		slog.WarnContext(r.Context(), "Couldn't get sessions", slog.Any("err", err))
		rt.statusPage(w, r, 500, err.Error())
		return
	}
	rt.renderSessions(w, r, user, sessions, nil)
}

func (rt *Web) selfSessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.userSessionsPage(w, r, user.UserFull(r))
	}
}

func (rt *Web) userSessions() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userFull, err := rt.base.UserFullByName(r.Context(), strings.TrimSpace(r.PathValue("user")))
		if err != nil && !errors.Is(err, kilonova.ErrNotFound) {
			slog.WarnContext(r.Context(), "Could not get user", slog.Any("err", err))
			rt.statusPage(w, r, 500, "")
			return
		}
		if userFull == nil {
			rt.statusPage(w, r, 404, "")
			return
		}
		// Only admins and that specific user can view their sessions
		if !(user.UserBrief(r).IsAdmin() || user.UserBrief(r).ID == userFull.ID) {
			rt.statusPage(w, r, 403, "")
			return
		}

		rt.userSessionsPage(w, r, userFull)
	}
}

func (rt *Web) sessionsFilter() http.HandlerFunc {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	type filterQuery struct {
		ID         *string `json:"id"`
		UserID     *int    `json:"user_id"`
		UserPrefix *string `json:"user_prefix"`

		IPAddr   *string `json:"ip_addr"`
		IPPrefix *string `json:"ip_prefix"`

		Page int `json:"page"`

		Ordering  string `json:"ord"`
		Ascending bool   `json:"asc"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		var q filterQuery
		r.ParseForm()
		if err := decoder.Decode(&q, r.Form); err != nil {
			slog.WarnContext(r.Context(), "Couldn't decode filter query", slog.Any("err", err))
		}
		if q.Page < 1 {
			q.Page = 1
		}

		f := &sudoapi.SessionFilter{
			ID: q.ID, UserID: q.UserID, UserPrefix: q.UserPrefix,

			Limit: 50, Offset: (q.Page - 1) * 50,
			Ordering: q.Ordering, Ascending: q.Ascending,
		}

		if q.IPAddr != nil {
			addr, err := netip.ParseAddr(*q.IPAddr)
			if err != nil {
				rt.statusPage(w, r, 400, "Invalid IP address: "+err.Error())
				return
			}
			f.IPAddr = &addr
		}
		if q.IPPrefix != nil {
			prefix, err := netip.ParsePrefix(*q.IPPrefix)
			if err != nil {
				rt.statusPage(w, r, 400, "Invalid IP prefix: "+err.Error())
				return
			}
			f.IPPrefix = &prefix
		}

		sessions, err := rt.base.Sessions(r.Context(), f)
		if err != nil {
			rt.statusPage(w, r, 500, err.Error())
			return
		}

		numSessions, err := rt.base.CountSessions(r.Context(), f)
		if err != nil {
			rt.statusPage(w, r, 500, err.Error())
			return
		}

		rt.renderSessions(w, r, nil, sessions, pageNav(q.Page, numSessions, 50))
	}
}
