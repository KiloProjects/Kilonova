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
	"go.uber.org/zap"
)

type UserBrief = kilonova.UserBrief
type UserFull = kilonova.UserFull

type Submissions struct {
	Submissions []*kilonova.Submission    `json:"submissions"`
	Count       int                       `json:"count"`
	Users       map[int]*UserBrief        `json:"users"`
	Problems    map[int]*kilonova.Problem `json:"problems"`
}

type BaseAPI struct {
	db      *db.DB
	manager kilonova.DataStore
	mailer  kilonova.Mailer
	rd      kilonova.MarkdownRenderer

	grader interface{ Wake() }

	logChan chan *logEntry
}

func (s *BaseAPI) Start(ctx context.Context) *StatusError {
	go s.ingestAuditLogs(ctx)
	go s.refreshProblemStatsJob(ctx, 5*time.Minute)

	return nil
}

func (s *BaseAPI) Close() *StatusError {
	if err := s.db.Close(); err != nil {
		return WrapError(err, "Couldn't close DB")
	}

	return nil
}

func GetBaseAPI(db *db.DB, manager kilonova.DataStore, mailer kilonova.Mailer) *BaseAPI {
	return &BaseAPI{db, manager, mailer, mdrenderer.NewLocalRenderer(), nil, make(chan *logEntry, 50)}
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

	mailer, err := email.NewMailer()
	if err != nil {
		return nil, WrapError(err, "Couldn't initialize mailer")
	}

	// DB Initialization
	db, err := db.NewPSQL(ctx, config.Common.DBDSN)
	if err != nil {
		return nil, WrapError(err, "Couldn't connect to DB")
	}
	zap.S().Info("Connected to DB")

	return GetBaseAPI(db, manager, mailer), nil
}
