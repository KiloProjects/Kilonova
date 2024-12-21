package sudoapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/natefinch/lumberjack.v2"
)

type logLevel int

const (
	logLevelImportant logLevel = iota
	logLevelDiscord
	logLevelWarning
	logLevelInfo
	logLevelVerbose
)

var (
	ImportantUpdatesWebhook = config.GenFlag[string]("admin.important_webhook", "", "Webhook URL for audit log-level events")
	VerboseUpdatesWebhook   = config.GenFlag[string]("admin.verbose_webhook", "", "Webhook URL for verbose platform information")

	EmailBranding = config.GenFlag("admin.mailer.branding", "Kilonova", "Branding to use at the end of emails")
)

func (s *BaseAPI) ResetWaitingSubmissions(ctx context.Context) error {
	subs, err := s.db.Submissions(ctx, kilonova.SubmissionFilter{Status: kilonova.StatusWorking})
	if err != nil {
		return fmt.Errorf("Couldn't get submissions to reset: %w", err)
	}
	ids := make([]int, 0, len(subs))
	for _, sub := range subs {
		ids = append(ids, sub.ID)
	}
	if err := s.db.ResetSubmissions(ctx, kilonova.SubmissionFilter{IDs: ids}); err != nil {
		slog.WarnContext(ctx, "Couldn't reset submissions", slog.Any("err", err))
		return fmt.Errorf("Couldn't reset submissions: %w", err)
	}

	// Wake grader to start processing immediately
	s.WakeGrader()
	return nil
}

func (s *BaseAPI) ResetSubmission(ctx context.Context, id int) error {
	if err := s.db.ResetSubmissions(ctx, kilonova.SubmissionFilter{ID: &id}); err != nil {
		slog.WarnContext(ctx, "Couldn't reset submission", slog.Any("err", err))
		return Statusf(500, "Couldn't reset submission")
	}

	// Wake grader to start processing immediately
	s.WakeGrader()

	return nil
}

func (s *BaseAPI) SetAdmin(ctx context.Context, user *kilonova.UserBrief, toSet bool) error {
	if !toSet {
		if user.ID == 1 {
			return Statusf(406, "First user must have admin rights.")
		}
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user to admin status", slog.Any("user", user))
	} else {
		s.LogUserAction(ctx, "Demoted user from admin status", slog.Any("user", user))
	}

	return s.updateUser(ctx, user.ID, kilonova.UserFullUpdate{Admin: &toSet, Proposer: &toSet})
}

func (s *BaseAPI) SetProposer(ctx context.Context, user *kilonova.UserBrief, toSet bool) error {
	if user.Admin {
		return Statusf(400, "Cannot update proposer status of an admin.")
	}

	if toSet {
		s.LogUserAction(ctx, "Promoted user to proposer status", slog.Any("user", user))
	} else {
		s.LogUserAction(ctx, "Demoted user from proposer status", slog.Any("user", user))
	}

	return s.updateUser(ctx, user.ID, kilonova.UserFullUpdate{Proposer: &toSet})
}

func (s *BaseAPI) SendMail(ctx context.Context, msg *kilonova.MailerMessage) error {
	if !s.MailerEnabled() {
		return Statusf(http.StatusServiceUnavailable, "Mailer is disabled")
	}
	if err := s.mailer.SendEmail(ctx, msg); err != nil {
		return fmt.Errorf("Could not send mail: %w", err)
	}
	return nil
}

// Warms up markdown statement cache by force triggering renders
func (s *BaseAPI) WarmupStatementCache(ctx context.Context) error {
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
			if _, err := s.RenderedProblemDesc(ctx, pb, variant); err != nil {
				return err
			}
		}
	}
	slog.InfoContext(ctx, "Triggered statement cache", slog.Duration("duration", time.Since(start)))
	s.LogUserAction(ctx, "Triggered statement cache warmup")
	return nil
}

type logEntry struct {
	Message string
	Attrs   []slog.Attr
	Author  *kilonova.UserBrief

	Level logLevel
}

func (s *logEntry) Equal(other *logEntry) bool {
	if s.Message != other.Message || s.Level != other.Level {
		return false
	}
	if (s.Author == nil) != (other.Author == nil) { // check if both either have or don't have an author
		return false
	}
	if s.Author != nil && s.Author.ID != other.Author.ID {
		return false
	}
	return equalAttrs(s.Attrs, other.Attrs)
}

func (s *BaseAPI) logAction(ctx context.Context, level logLevel, msg string, args []slog.Attr) {
	s.logChan <- &logEntry{
		Message: msg,
		Attrs:   args,
		Author:  util.UserBriefContext(ctx),
		Level:   level,
	}
}

func (s *BaseAPI) LogToDiscord(ctx context.Context, msg string, args ...slog.Attr) {
	s.logAction(ctx, logLevelDiscord, msg, args)
}

func (s *BaseAPI) LogUserAction(ctx context.Context, msg string, args ...slog.Attr) {
	s.logAction(ctx, logLevelImportant, msg, args)
}

func (s *BaseAPI) LogInfo(ctx context.Context, msg string, attrs ...slog.Attr) {
	s.logAction(ctx, logLevelInfo, msg, attrs)
}

func (s *BaseAPI) LogVerbose(ctx context.Context, msg string, args ...slog.Attr) {
	s.logAction(ctx, logLevelVerbose, msg, args)
}

func (s *BaseAPI) GetAuditLogs(ctx context.Context, count int, offset int) ([]*kilonova.AuditLog, error) {
	logs, err := s.db.AuditLogs(ctx, count, offset)
	if err != nil {
		return nil, fmt.Errorf("Couldn't fetch audit logs: %w", err)
	}
	return logs, nil
}

func (s *BaseAPI) GetLogCount(ctx context.Context) (int, error) {
	cnt, err := s.db.AuditLogCount(ctx)
	if err != nil {
		return -1, fmt.Errorf("Couldn't get audit log count: %w", err)
	}
	return cnt, nil
}

type webhookSender struct {
	lastMessageEntry *logEntry
	lastMessageID    string
	lastMessageCount int

	webhookID    string
	webhookToken string

	name string
	mu   sync.Mutex

	base *BaseAPI
}

func (ws *webhookSender) getWebhookEmbed(entry *logEntry, showRepeatCount bool) *discordgo.MessageEmbed {
	hostPrefix := config.Common.HostPrefix
	embed := &discordgo.MessageEmbed{
		Title:       entry.Message,
		Description: "New action",
		Fields:      make([]*discordgo.MessageEmbedField, 0, len(entry.Attrs)+2),
	}
	if entry.Author != nil {
		embed.Description += " (by [" + entry.Author.AppropriateName() + "](" + hostPrefix + "/profile/" + entry.Author.Name + ")"
		if entry.Author.DiscordID != nil {
			embed.Description += " / <@" + *entry.Author.DiscordID + ">"
		}

		embed.Description += ")"
	}
	cc := cases.Title(language.English)
	for _, attr := range entry.Attrs {
		val := attr.Value.String()
		switch v := attr.Value.Any().(type) {
		case *kilonova.UserBrief:
			var dID string
			if v.DiscordID != nil {
				dID = " / <@" + *v.DiscordID + ">"
			}
			val = fmt.Sprintf("[%s (#%d)](%s/profile/%s)", v.AppropriateName(), v.ID, hostPrefix, v.Name) + dID
		case *kilonova.Problem:
			val = fmt.Sprintf("[%s (#%d)](%s/problems/%d)", v.Name, v.ID, hostPrefix, v.ID)
		case *kilonova.Contest:
			val = fmt.Sprintf("[%s (#%d)](%s/contests/%d)", v.Name, v.ID, hostPrefix, v.ID)
		case *kilonova.Tag:
			val = fmt.Sprintf("[%s (#%d)](%s/tags/%d)", v.Name, v.ID, hostPrefix, v.ID)
		case *kilonova.BlogPost:
			val = fmt.Sprintf("[%s (#%d)](%s/posts/%s)", v.Title, v.ID, hostPrefix, v.Slug)
		case *kilonova.ProblemList:
			val = fmt.Sprintf("[%s (#%d)](%s/problem_lists/%d)", v.Title, v.ID, hostPrefix, v.ID)
		case slog.LogValuer:
			val = v.LogValue().String()
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   cc.String(strings.ReplaceAll(attr.Key, "_", " ")),
			Value:  val,
			Inline: true,
		})
	}
	if showRepeatCount {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: "Action repeated " + strconv.Itoa(ws.lastMessageCount) + " times",
		}
	}

	return embed
}

func (ws *webhookSender) editLastMessage() error {
	ws.lastMessageCount++
	if _, err := ws.base.dSess.WebhookMessageEdit(ws.webhookID, ws.webhookToken, ws.lastMessageID, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{
			ws.getWebhookEmbed(ws.lastMessageEntry, true),
		},
	}); err != nil {
		return fmt.Errorf("Couldn't edit webhook message: %w", err)
	}
	return nil
}

func (ws *webhookSender) Send(ctx context.Context, entry *logEntry) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.lastMessageEntry != nil && ws.lastMessageEntry.Equal(entry) {
		if err := ws.editLastMessage(); err == nil {
			return nil
		} else {
			slog.WarnContext(ctx, "Could not edit last webhook message. Defaulting to sending a new one", slog.Any("err", err))
		}
	}

	msg, err := ws.base.dSess.WebhookExecute(ws.webhookID, ws.webhookToken, true, &discordgo.WebhookParams{
		Username: ws.name,

		Embeds: []*discordgo.MessageEmbed{
			ws.getWebhookEmbed(entry, false),
		},
	})
	if err != nil {
		slog.WarnContext(ctx, "Unsuccessful Webhook execution", slog.Any("err", err), slog.Any("entry", entry))
		return fmt.Errorf("Couldn't execute webhook: %w", err)
	}

	if msg != nil {
		ws.lastMessageID = msg.ID
		ws.lastMessageEntry = entry
		ws.lastMessageCount = 1
		return nil
	}

	slog.DebugContext(ctx, "Empty message response", slog.Any("entry", entry))
	ws.lastMessageID = ""
	ws.lastMessageEntry = nil
	ws.lastMessageCount = -1
	return nil
}

func (s *BaseAPI) newWebhookSender(ctx context.Context, webhookURL string, name string) *webhookSender {
	if webhookURL == "" {
		return nil
	}
	url, err := url.Parse(webhookURL)
	if err != nil {
		slog.WarnContext(ctx, "Invalid webhook URL", slog.Any("err", err))
		return nil
	}
	parts := strings.Split(url.Path, "/")
	if len(parts) < 2 {
		slog.WarnContext(ctx, "Invalid webhook URL", slog.Any("err", err))
		return nil
	}
	return &webhookSender{
		webhookID:    parts[len(parts)-2],
		webhookToken: parts[len(parts)-1],

		name: name,

		base: s,
	}
}

func (s *BaseAPI) processLogEntry(ctx context.Context, val *logEntry, importantWebhook, verboseWebhook *webhookSender) {
	defer func() {
		if err := recover(); err != nil {
			slog.WarnContext(ctx, "Log entry panic", slog.Any("err", err))
		}
	}()
	var id *int
	if val.Author != nil {
		id = &val.Author.ID
	}

	if val.Level.IsAuditLogLvl() && val.Level != logLevelDiscord {
		attrs, err := json.Marshal(marshalAttrs(val.Attrs...))
		if err != nil {
			attrs = []byte(`{"attrs": "err"}`)
		}
		if _, err := s.db.CreateAuditLog(ctx, val.Message+" "+string(attrs), id, false); err != nil {
			zap.S().Warn("Couldn't store audit log entry to database: ", err)
		}
	}

	if val.Level != logLevelDiscord {
		var s strings.Builder
		if val.Author != nil {
			s.WriteString(fmt.Sprintf(" (by user #%d: %s)", val.Author.ID, val.Author.Name))
		}
		s.WriteString(": " + val.Message)
		slog.LogAttrs(ctx, val.Level.toSlog(), s.String(), val.Attrs...)
	}

	if val.Level.IsAuditLogLvl() && importantWebhook != nil {
		if err := importantWebhook.Send(ctx, val); err != nil {
			slog.WarnContext(ctx, "Could not send to important webhook", slog.Any("err", err))
		}
	}

	if !val.Level.IsAuditLogLvl() && verboseWebhook != nil {
		if err := verboseWebhook.Send(ctx, val); err != nil {
			slog.WarnContext(ctx, "Could not send to verbose webhook", slog.Any("err", err))
		}
	}
}

func (s *BaseAPI) ingestAuditLogs(ctx context.Context) error {
	importantWebhook := s.newWebhookSender(ctx, ImportantUpdatesWebhook.Value(), "Kilonova Audit Log")
	verboseWebhook := s.newWebhookSender(ctx, VerboseUpdatesWebhook.Value(), "Kilonova Verbose Log")
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case val := <-s.logChan:
			s.processLogEntry(ctx, val, importantWebhook, verboseWebhook)
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
	go s.cleanupBuckets(ctx)
	for {
		select {
		case <-ctx.Done():
			if !errors.Is(ctx.Err(), context.Canceled) {
				return ctx.Err()
			}
			return nil
		case <-t.C:
			slog.DebugContext(ctx, "Running eviction policy")
			s.cleanupBuckets(ctx)
		}
	}
}

func (s *BaseAPI) EvictionLogger() *slog.Logger { return s.evictionLogger }

func (s *BaseAPI) cleanupBuckets(ctx context.Context) {
	for _, bucket := range datastore.GetBuckets() {
		if !bucket.Evictable() {
			continue
		}
		s.evictionLogger.InfoContext(ctx, "Running bucket eviction policy", slog.Any("bucket", bucket))
		numDeleted, err := bucket.RunEvictionPolicy(ctx, s.evictionLogger)
		if err != nil {
			s.evictionLogger.ErrorContext(ctx, "Could not run eviction policy", slog.Any("err", err))
			slog.WarnContext(ctx, "Error running bucket cleanup. Check eviction.log for details")
			continue
		}
		s.evictionLogger.InfoContext(ctx, "Deleted bucket objects", slog.Any("bucket", bucket), slog.Int("count", numDeleted))
	}
}

func (s *BaseAPI) refreshProblemStatsJob(ctx context.Context, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	go func() {
		// Initial refresh
		slog.DebugContext(ctx, "Refreshing problem statistics")
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
			slog.DebugContext(ctx, "Refreshing problem statistics")
			s.db.RefreshProblemStats(ctx)
		}
	}
}

func (s *BaseAPI) refreshHotProblemsJob(ctx context.Context, interval time.Duration) error {
	t := time.NewTicker(interval)
	defer t.Stop()
	go func() {
		// Initial refresh
		slog.DebugContext(ctx, "Refreshing hot problems")
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
			slog.DebugContext(ctx, "Refreshing hot problems")
			s.db.RefreshHotProblems(ctx, config.Frontend.BannedHotProblems)
		}
	}
}

func (s *BaseAPI) LanguageVersions(ctx context.Context) map[string]string {
	return s.grader.LanguageVersions(ctx)
}

func (s *BaseAPI) WakeGrader() {
	if s.grader != nil {
		s.grader.Wake()
	}
}

func (s *BaseAPI) RegisterGrader(gr Grader) {
	s.grader = gr
	// Initial load
	go s.LanguageVersions(context.Background())
}

func (ll logLevel) IsAuditLogLvl() bool {
	return ll == logLevelImportant || ll == logLevelWarning || ll == logLevelDiscord
}

func (ll logLevel) toSlog() slog.Level {
	switch ll {
	case logLevelWarning:
		return slog.LevelWarn
	case logLevelVerbose:
		return slog.LevelDebug
	default:
		return slog.LevelInfo
	}
}

func equalAttrs(a, b []slog.Attr) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Key != b[i].Key || a[i].Value.Kind() != b[i].Value.Kind() {
			return false
		}
		switch a[i].Value.Kind() {
		case slog.KindLogValuer:
			// Try to go deeper
			if !equalAttrs(
				[]slog.Attr{{Key: "deep", Value: a[i].Value.LogValuer().LogValue()}},
				[]slog.Attr{{Key: "deep", Value: b[i].Value.LogValuer().LogValue()}},
			) {
				return false
			}
		case slog.KindAny:
			if !reflect.DeepEqual(a[i].Value.Any(), b[i].Value.Any()) {
				return false
			}
		case slog.KindGroup:
			if !equalAttrs(a[i].Value.Group(), b[i].Value.Group()) {
				return false
			}
		default:
			if !a[i].Value.Equal(b[i].Value) {
				return false
			}
		}
	}

	return true
}

func marshalAttr(attr slog.Attr) any {
	switch attr.Value.Kind() {
	case slog.KindGroup:
		return marshalAttrs(attr.Value.Group()...)
	case slog.KindLogValuer:
		return marshalAttr(slog.Attr{Key: "deep", Value: attr.Value.LogValuer().LogValue()})
	default:
		return attr.Value.String()
	}
}

func marshalAttrs(a ...slog.Attr) map[string]any {
	var attMap = make(map[string]any)
	for _, attr := range a {
		attMap[attr.Key] = marshalAttr(attr)
	}
	return attMap
}
