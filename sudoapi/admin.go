package sudoapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logLevel int

const (
	logLevelSystem logLevel = iota
	logLevelImportant
	logLevelDiscord
	logLevelWarning
	logLevelInfo
	logLevelVerbose
)

var (
	ImportantUpdatesWebhook = config.GenFlag[string]("admin.important_webhook", "", "Webhook URL for audit log-level events")
	VerboseUpdatesWebhook   = config.GenFlag[string]("admin.verbose_webhook", "", "Webhook URL for verbose platform information")
)

func (s *BaseAPI) ResetWaitingSubmissions(ctx context.Context) *StatusError {
	subs, err := s.db.Submissions(ctx, kilonova.SubmissionFilter{Status: kilonova.StatusWorking})
	if err != nil {
		return WrapError(err, "Couldn't get submissions to reset")
	}
	ids := make([]int, 0, len(subs))
	for _, sub := range subs {
		ids = append(ids, sub.ID)
	}
	if err := s.db.ResetSubmissions(ctx, kilonova.SubmissionFilter{IDs: ids}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) ResetSubmission(ctx context.Context, id int) *StatusError {
	if err := s.db.ResetSubmissions(ctx, kilonova.SubmissionFilter{ID: &id}); err != nil {
		zap.S().Warn("Couldn't reset submission: ", err)
		return Statusf(500, "Couldn't reset submission")
	}

	// Wake grader to start processing immediately
	s.WakeGrader()

	return nil
}

func (s *BaseAPI) SetAdmin(ctx context.Context, userID int, toSet bool) *StatusError {
	if userID <= 0 {
		return Statusf(400, "Invalid ID")
	}

	if !toSet {
		if userID == 1 {
			return Statusf(406, "First user must have admin rights.")
		}
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user #%d to admin status", userID)
	} else {
		s.LogUserAction(ctx, "Demoted user #%d from admin status", userID)
	}

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Admin: &toSet, Proposer: &toSet})
}

func (s *BaseAPI) SetProposer(ctx context.Context, userID int, toSet bool) *StatusError {
	user, err := s.UserBrief(ctx, userID)
	if err != nil {
		return err
	}

	if user.Admin {
		return Statusf(400, "Cannot update proposer status of an admin.")
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user #%d to proposer status", userID)
	} else {
		s.LogUserAction(ctx, "Demoted user #%d from proposer status", userID)
	}

	return s.updateUser(ctx, userID, kilonova.UserFullUpdate{Proposer: &toSet})
}

func (s *BaseAPI) SendMail(msg *kilonova.MailerMessage) *StatusError {
	if !s.MailerEnabled() {
		return Statusf(http.StatusServiceUnavailable, "Mailer is disabled")
	}
	if err := s.mailer.SendEmail(msg); err != nil {
		return WrapError(err, "Could not send mail")
	}
	return nil
}

// Warms up markdown statement cache by force triggering renders
func (s *BaseAPI) WarmupStatementCache(ctx context.Context) *StatusError {
	start := time.Now()
	pbs, err := s.Problems(ctx, kilonova.ProblemFilter{})
	if err != nil {
		return err
	}
	for _, pb := range pbs {
		variants, err := s.ProblemDescVariants(ctx, pb.ID, true)
		if err != nil {
			return err
		}
		for _, variant := range variants {
			if variant.Format != "md" {
				continue
			}
			if _, err := s.RenderedProblemDesc(ctx, pb, variant.Language, variant.Format, variant.Type); err != nil {
				return err
			}
		}
	}
	slog.Info("Triggered statement cache", slog.Duration("duration", time.Since(start)))
	s.LogUserAction(ctx, "Triggered statement cache warmup")
	return nil
}

type logEntry struct {
	Message string
	Author  *kilonova.UserBrief

	Level logLevel
}

func (s *BaseAPI) logAction(ctx context.Context, level logLevel, msg string, args ...any) {
	s.logChan <- &logEntry{
		Message: fmt.Sprintf(msg, args...),
		Author:  util.UserBriefContext(ctx),
		Level:   level,
	}
}

func (s *BaseAPI) LogSystemAction(ctx context.Context, msg string, args ...any) {
	s.logAction(ctx, logLevelSystem, msg, args...)
}

func (s *BaseAPI) LogToDiscord(ctx context.Context, msg string, args ...any) {
	s.logAction(ctx, logLevelDiscord, msg, args...)
}

func (s *BaseAPI) LogUserAction(ctx context.Context, msg string, args ...any) {
	s.logAction(ctx, logLevelImportant, msg, args...)
}

func (s *BaseAPI) LogInfo(ctx context.Context, msg string, args ...any) {
	s.logAction(ctx, logLevelInfo, msg, args...)
}

func (s *BaseAPI) LogVerbose(ctx context.Context, msg string, args ...any) {
	s.logAction(ctx, logLevelVerbose, msg, args...)
}

func (s *BaseAPI) GetAuditLogs(ctx context.Context, count int, offset int) ([]*kilonova.AuditLog, *StatusError) {
	logs, err := s.db.AuditLogs(ctx, count, offset)
	if err != nil {
		return nil, WrapError(err, "Couldn't fetch audit logs")
	}
	return logs, nil
}

func (s *BaseAPI) GetLogCount(ctx context.Context) (int, *StatusError) {
	cnt, err := s.db.AuditLogCount(ctx)
	if err != nil {
		return -1, WrapError(err, "Couldn't get audit log count")
	}
	return cnt, nil
}

type webhookSender struct {
	lastMessageText  string
	lastMessageID    string
	lastMessageCount int

	webhookID    string
	webhookToken string

	name string
	mu   sync.Mutex
}

func (ws *webhookSender) EditLastMessage(ctx context.Context) *StatusError {
	vals := make(url.Values)
	vals.Add("content", fmt.Sprintf("%s (message repeated %d times)", ws.lastMessageText, ws.lastMessageCount+1))
	req, err := http.NewRequestWithContext(ctx, "PATCH", fmt.Sprintf("https://discord.com/api/webhooks/%s/%s/messages/%s", ws.webhookID, ws.webhookToken, ws.lastMessageID), strings.NewReader(vals.Encode()))
	if err != nil {
		return WrapError(err, "Couldn't build request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return WrapError(err, "Couldn't execute request")
	}
	resp.Body.Close()

	ws.lastMessageCount++
	return nil
}

func (ws *webhookSender) Send(ctx context.Context, text string) *StatusError {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if text == ws.lastMessageText {
		return ws.EditLastMessage(ctx)
	}

	vals := make(url.Values)
	vals.Add("content", text)
	vals.Add("username", ws.name)
	// We have to wait in order to see message ID
	req, err := http.NewRequestWithContext(ctx, "POST", fmt.Sprintf("https://discord.com/api/webhooks/%s/%s?wait=true", ws.webhookID, ws.webhookToken), strings.NewReader(vals.Encode()))
	if err != nil {
		return WrapError(err, "Couldn't build request")
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil
		}
		return WrapError(err, "Couldn't execute request")
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 && resp.Header.Get("Content-Type") == "application/json" {
		var message = make(map[string]any)
		if err := json.NewDecoder(resp.Body).Decode(&message); err != nil {
			zap.S().Warn("Invalid JSON from Discord")
		}
		switch v := message["id"].(type) {
		case string:
			ws.lastMessageID = v
			ws.lastMessageText = text
			ws.lastMessageCount = 1
			return nil
		default:
			zap.S().Warn("Invalid webhook message ID from Discord: ", v)
		}
	} else {
		val, _ := io.ReadAll(resp.Body)
		spew.Dump(resp.Header)
		zap.S().Warn("Unsuccessful Discord request: ", string(val))
	}
	ws.lastMessageID = ""
	ws.lastMessageText = ""
	ws.lastMessageCount = -1
	return nil
}

func newWebhookSender(webhookURL string, name string) *webhookSender {
	if webhookURL == "" {
		return nil
	}
	url, err := url.Parse(webhookURL)
	if err != nil {
		zap.S().Warn("Invalid webhook URL: ", err)
		return nil
	}
	parts := strings.Split(url.Path, "/")
	if len(parts) < 2 {
		zap.S().Warn("Invalid webhook URL: ", err)
		return nil
	}
	return &webhookSender{
		webhookID:    parts[len(parts)-2],
		webhookToken: parts[len(parts)-1],

		name: name,
	}
}

func (s *BaseAPI) ingestAuditLogs(ctx context.Context) error {
	importantWebhook := newWebhookSender(ImportantUpdatesWebhook.Value(), "Kilonova Audit Log")
	verboseWebhook := newWebhookSender(VerboseUpdatesWebhook.Value(), "Kilonova Verbose Log")
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case val := <-s.logChan:
			var id *int
			if val.Author != nil {
				id = &val.Author.ID
			}

			if val.Level.IsAuditLogLvl() && val.Level != logLevelDiscord {
				if _, err := s.db.CreateAuditLog(ctx, val.Message, id, val.Level == logLevelSystem); err != nil {
					zap.S().Warn("Couldn't store audit log entry to database: ", err)
				}
			}

			var s strings.Builder
			s.WriteString("Action")
			if val.Level == logLevelSystem {
				s.WriteString(" (system)")
			} else if val.Author != nil {
				s.WriteString(fmt.Sprintf(" (by user #%d: %s)", val.Author.ID, val.Author.Name))
			}
			s.WriteString(": " + val.Message)

			if val.Level != logLevelDiscord {
				zap.S().Desugar().Log(val.Level.toZap(), s.String())
			}

			if val.Level.IsAuditLogLvl() && importantWebhook != nil {
				if err := importantWebhook.Send(ctx, s.String()); err != nil {
					zap.S().Warn(err)
				}
			}

			if !val.Level.IsAuditLogLvl() && verboseWebhook != nil {
				if err := verboseWebhook.Send(ctx, s.String()); err != nil {
					zap.S().Warn(err)
				}
			}
		}
	}
}

func (s *BaseAPI) cleanupBucketsJob(ctx context.Context, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	logFile := &lumberjack.Logger{
		Filename: path.Join(config.Common.LogDir, "eviction.log"),
		MaxSize:  80, //MB
		Compress: true,
	}
	lvl := slog.LevelInfo
	if config.Common.Debug {
		lvl = slog.LevelDebug
	}
	s.evictionLogger = slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		AddSource: true,
		Level:     lvl,
	}))
	// Initial refresh
	go s.cleanupBuckets()
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case <-t.C:
			zap.S().Debug("Running eviction policy")
			s.cleanupBuckets()
		}
	}
}

func (s *BaseAPI) EvictionLogger() *slog.Logger { return s.evictionLogger }

func (s *BaseAPI) cleanupBuckets() {
	for _, bucket := range datastore.GetBuckets() {
		if !bucket.Evictable() {
			continue
		}
		s.evictionLogger.Info("Running bucket eviction policy", slog.Any("bucket", bucket))
		numDeleted, err := bucket.RunEvictionPolicy(s.evictionLogger)
		if err != nil {
			s.evictionLogger.Error(err.Error())
			zap.S().Warn("Error running bucket cleanup. Check eviction.log for details")
			continue
		}
		s.evictionLogger.Info("Deleted bucket objects", slog.Any("bucket", bucket), slog.Int("count", numDeleted))
	}
}

func (s *BaseAPI) refreshProblemStatsJob(ctx context.Context, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	go func() {
		// Initial refresh
		zap.S().Debug("Refreshing problem statistics")
		s.db.RefreshProblemStats(ctx)
	}()
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case <-t.C:
			zap.S().Debug("Refreshing problem statistics")
			s.db.RefreshProblemStats(ctx)
		}
	}
}

func (s *BaseAPI) refreshHotProblemsJob(ctx context.Context, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	go func() {
		// Initial refresh
		zap.S().Debug("Refreshing hot problems")
		s.db.RefreshHotProblems(ctx, config.Frontend.BannedHotProblems)
	}()
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case <-t.C:
			zap.S().Debug("Refreshing hot problems")
			s.db.RefreshHotProblems(ctx, config.Frontend.BannedHotProblems)
		}
	}
}

func (s *BaseAPI) WakeGrader() {
	if s.grader != nil {
		s.grader.Wake()
	}
}

func (s *BaseAPI) RegisterGrader(gr interface{ Wake() }) {
	s.grader = gr
}

func (ll logLevel) IsAuditLogLvl() bool {
	return ll == logLevelSystem || ll == logLevelImportant || ll == logLevelWarning || ll == logLevelDiscord
}

func (ll logLevel) toZap() zapcore.Level {
	switch ll {
	case logLevelWarning:
		return zap.WarnLevel
	case logLevelVerbose:
		return zap.DebugLevel
	default:
		return zapcore.InfoLevel
	}
}
