package sudoapi

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/email"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer"
	"github.com/Yiling-J/theine-go"
	"github.com/bwmarrin/discordgo"
	"github.com/zitadel/oidc/v3/pkg/op"
)

type Submissions struct {
	Submissions []*kilonova.Submission `json:"submissions"`
	Count       int                    `json:"count"`
	Truncated   bool                   `json:"truncated_count"`

	Users    map[int]*kilonova.UserBrief `json:"users"`
	Problems map[int]*kilonova.Problem   `json:"problems"`
}

type Grader interface {
	Wake()
	Language(string) *eval.Language
	Languages() map[string]*eval.Language
	LanguageVersions(ctx context.Context) map[string]string
}

type BaseAPI struct {
	db     *db.DB
	mailer kilonova.Mailer
	rd     *mdrenderer.Renderer

	mgr *datastore.Manager

	sessionUserCache *theine.LoadingCache[string, *kilonova.UserFull]

	grader Grader

	logChan chan *logEntry

	dSess *discordgo.Session

	evictionLogger        *slog.Logger
	testBucket            datastore.Bucket
	attachmentCacheBucket datastore.Bucket
	subtestBucket         datastore.Bucket
	avatarBucket          datastore.Bucket

	oidcProvider *op.Provider
}

func (s *BaseAPI) Start(ctx context.Context) {
	if err := s.initDiscord(ctx); err != nil {
		slog.WarnContext(ctx, "Could not initialize Discord", slog.Any("err", err))
	}
	go s.ingestAuditLogs(ctx)
	go s.cleanupBucketsJob(ctx, 30*time.Minute)
	go s.refreshProblemStatsJob(ctx, 5*time.Minute)
	go s.refreshHotProblemsJob(ctx, 4*time.Hour)
}

func (s *BaseAPI) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("couldn't close DB: %w", err)
	}

	if s.dSess != nil {
		if err := s.dSess.Close(); err != nil {
			return fmt.Errorf("couldn't close Discord session: %w", err)
		}
	}

	return nil
}

func GetBaseAPI(ctx context.Context, db *db.DB, mgr *datastore.Manager, mailer kilonova.Mailer) (*BaseAPI, error) {
	base := &BaseAPI{
		db:     db,
		mailer: mailer,
		rd:     mdrenderer.NewRenderer(),

		mgr: mgr,

		sessionUserCache: nil,

		grader:  nil,
		logChan: make(chan *logEntry, 50),

		testBucket:            mgr.Tests(),
		attachmentCacheBucket: mgr.Attachments(),
		subtestBucket:         mgr.Subtests(),
		avatarBucket:          mgr.Avatars(),
	}
	sUserCache, err := theine.NewBuilder[string, *kilonova.UserFull](500).BuildWithLoader(func(ctx context.Context, sid string) (theine.Loaded[*kilonova.UserFull], error) {
		user, err := base.sessionUser(ctx, sid)
		if err != nil {
			return theine.Loaded[*kilonova.UserFull]{}, err
		}
		return theine.Loaded[*kilonova.UserFull]{
			Value: user,
			Cost:  1,
			TTL:   20 * time.Second,
		}, nil
	})
	if err != nil {
		return nil, fmt.Errorf("could not build session user cache: %w", err)
	}
	base.sessionUserCache = sUserCache

	storage := auth.NewAuthStorage(ctx, base.db.GetPool())
	provider, err := auth.GetProvider(storage)
	if err != nil {
		return nil, err
	}
	base.oidcProvider = provider
	return base, nil
}

func InitializeBaseAPI(ctx context.Context) (*BaseAPI, error) {
	// Data directory setup
	if !path.IsAbs(config.Common.DataDir) {
		return nil, Statusf(400, "dataDir is not absolute")
	}
	if err := os.MkdirAll(config.Common.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("couldn't create data dir: %w", err)
	}

	mgr, err := datastore.New(config.Common.DataDir)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialize data store: %w", err)
	}

	var knMailer kilonova.Mailer
	if config.Email.Enabled {
		mailer, err := email.NewMailer()
		if err != nil {
			slog.WarnContext(ctx, "Couldn't initialize mailer. Make sure you entered the correct information", slog.Any("err", err))
		}
		knMailer = mailer
	}

	// DB Initialization
	dbClient, err := db.NewPSQL(ctx, config.Common.DBDSN)
	if err != nil {
		return nil, fmt.Errorf("couldn't connect to DB: %w", err)
	}
	slog.InfoContext(ctx, "Connected to DB")

	if flags.MigrateOnStart.Value() {
		if err := dbClient.RunMigrations(ctx); err != nil {
			return nil, fmt.Errorf("couldn't run migrations: %w", err)
		}
	}

	return GetBaseAPI(ctx, dbClient, mgr, knMailer)
}

func InitQueryCounter(ctx context.Context) context.Context {
	return db.InitContextCounter(ctx)
}

func GetQueryCounter(ctx context.Context) int64 {
	return db.GetContextQueryCount(ctx)
}
