package sudoapi

import (
	"context"
	"log/slog"
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/email"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer"
	"github.com/Yiling-J/theine-go"
	"github.com/bwmarrin/discordgo"
	"go.uber.org/zap"
)

var (
	MigrateOnStart = config.GenFlag("behavior.db.run_migrations", true, "Run PostgreSQL migrations on platform start")
)

type UserBrief = kilonova.UserBrief
type UserFull = kilonova.UserFull

type Submissions struct {
	Submissions []*kilonova.Submission `json:"submissions"`
	Count       int                    `json:"count"`
	Truncated   bool                   `json:"truncated_count"`

	Users    map[int]*UserBrief        `json:"users"`
	Problems map[int]*kilonova.Problem `json:"problems"`
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
	rd     kilonova.MarkdownRenderer

	sessionUserCache *theine.LoadingCache[string, *kilonova.UserFull]

	grader Grader

	logChan chan *logEntry

	dSess *discordgo.Session

	evictionLogger        *slog.Logger
	testBucket            *datastore.Bucket
	attachmentCacheBucket *datastore.Bucket
	subtestBucket         *datastore.Bucket
	avatarBucket          *datastore.Bucket
}

func (s *BaseAPI) Start(ctx context.Context) {
	if err := s.initDiscord(); err != nil {
		slog.Warn("Could not initialize Discord", slog.Any("err", err))
	}
	go s.ingestAuditLogs(ctx)
	go s.cleanupBucketsJob(ctx, 30*time.Minute)
	go s.refreshProblemStatsJob(ctx, 5*time.Minute)
	go s.refreshHotProblemsJob(ctx, 4*time.Hour)
}

func (s *BaseAPI) Close() *StatusError {
	if err := s.db.Close(); err != nil {
		return WrapError(err, "Couldn't close DB")
	}

	if s.dSess != nil {
		if err := s.dSess.Close(); err != nil {
			return WrapError(err, "Couldn't close Discord session ")
		}
	}

	return nil
}

func GetBaseAPI(db *db.DB, mailer kilonova.Mailer) (*BaseAPI, *StatusError) {
	base := &BaseAPI{
		db:     db,
		mailer: mailer,
		rd:     mdrenderer.NewLocalRenderer(),

		sessionUserCache: nil,

		grader:  nil,
		logChan: make(chan *logEntry, 50),

		testBucket:            datastore.GetBucket(datastore.BucketTypeTests),
		attachmentCacheBucket: datastore.GetBucket(datastore.BucketTypeAttachments),
		subtestBucket:         datastore.GetBucket(datastore.BucketTypeSubtests),
		avatarBucket:          datastore.GetBucket(datastore.BucketTypeAvatars),
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
		return nil, WrapError(err, "Could not build session user cache")
	}
	base.sessionUserCache = sUserCache
	return base, nil
}

func InitializeBaseAPI(ctx context.Context) (*BaseAPI, *StatusError) {
	// Data directory setup
	if !path.IsAbs(config.Common.DataDir) {
		return nil, Statusf(400, "dataDir is not absolute")
	}
	if err := os.MkdirAll(config.Common.DataDir, 0755); err != nil {
		return nil, WrapError(err, "Couldn't create data dir")
	}

	if err := datastore.InitBuckets(config.Common.DataDir); err != nil {
		return nil, WrapError(err, "Couldn't initialize data store")
	}

	var knMailer kilonova.Mailer
	if config.Email.Enabled {
		mailer, err := email.NewMailer()
		if err != nil {
			zap.S().Warn("Couldn't initialize mailer: ", err)
			zap.S().Warn("Make sure you entered the correct information")
		}
		knMailer = mailer
	}

	// DB Initialization
	db, err := db.NewPSQL(ctx, config.Common.DBDSN)
	if err != nil {
		return nil, WrapError(err, "Couldn't connect to DB")
	}
	zap.S().Info("Connected to DB")

	if MigrateOnStart.Value() {
		if err := db.RunMigrations(ctx); err != nil {
			return nil, WrapError(err, "Couldn't run migrations")
		}
	}

	return GetBaseAPI(db, knMailer)
}

func (s *BaseAPI) InitQueryCounter(ctx context.Context) context.Context {
	return db.InitContextCounter(ctx)
}

func (s *BaseAPI) GetQueryCounter(ctx context.Context) int64 {
	return db.GetContextQueryCount(ctx)
}
