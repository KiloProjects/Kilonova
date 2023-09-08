package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	if err := s.db.BulkUpdateSubmissions(ctx, kilonova.SubmissionFilter{Status: kilonova.StatusWorking}, kilonova.SubmissionUpdate{Status: kilonova.StatusWaiting}); err != nil {
		zap.S().Warn(err)
		return WrapError(err, "Couldn't reset submissions")
	}

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) ResetSubmission(ctx context.Context, id int) *StatusError {
	if err := s.db.ResetSubmission(ctx, id); err != nil {
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

func (s *BaseAPI) ingestAuditLogs(ctx context.Context) error {
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

			if val.Level.IsAuditLogLvl() {
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

			if val.Level.IsAuditLogLvl() && ImportantUpdatesWebhook.Value() != "" {
				vals := make(url.Values)
				vals.Add("content", s.String())
				vals.Add("username", "Kilonova Audit Log")
				_, err := http.PostForm(ImportantUpdatesWebhook.Value(), vals)
				if err != nil {
					zap.S().Warn(err)
				}
			}

			if !val.Level.IsAuditLogLvl() && VerboseUpdatesWebhook.Value() != "" {
				vals := make(url.Values)
				vals.Add("content", s.String())
				vals.Add("username", "Kilonova Verbose Log")
				_, err := http.PostForm(VerboseUpdatesWebhook.Value(), vals)
				if err != nil {
					zap.S().Warn(err)
				}
			}
		}
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

func (s *BaseAPI) WakeGrader() {
	if s.grader != nil {
		s.grader.Wake()
	}
}

func (s *BaseAPI) RegisterGrader(gr interface{ Wake() }) {
	s.grader = gr
}

func (ll logLevel) IsAuditLogLvl() bool {
	return ll == logLevelSystem || ll == logLevelImportant || ll == logLevelWarning
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
