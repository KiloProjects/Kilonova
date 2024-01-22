package sudoapi

import (
	"context"
	"os"
	"path"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/datastore"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/email"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer"
	"github.com/Yiling-J/theine-go"
	"go.uber.org/zap"
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

type BaseAPI struct {
	db      *db.DB
	manager kilonova.DataStore
	mailer  kilonova.Mailer
	rd      kilonova.MarkdownRenderer

	sessionUserCache *theine.LoadingCache[string, *kilonova.UserFull]

	grader interface{ Wake() }

	logChan chan *logEntry
}

func (s *BaseAPI) Start(ctx context.Context) {
	go s.ingestAuditLogs(ctx)
	go s.refreshProblemStatsJob(ctx, 5*time.Minute)
	go s.refreshHotProblemsJob(ctx, 4*time.Hour)
}

func (s *BaseAPI) Close() *StatusError {
	if err := s.db.Close(); err != nil {
		return WrapError(err, "Couldn't close DB")
	}

	return nil
}

func GetBaseAPI(db *db.DB, manager kilonova.DataStore, mailer kilonova.Mailer) (*BaseAPI, *StatusError) {
	base := &BaseAPI{
		db:      db,
		manager: manager,
		mailer:  mailer,
		rd:      mdrenderer.NewLocalRenderer(),

		sessionUserCache: nil,

		grader:  nil,
		logChan: make(chan *logEntry, 50),
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

	manager, err := datastore.NewManager(config.Common.DataDir)
	if err != nil {
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

	return GetBaseAPI(db, manager, knMailer)
}
