package sudoapi

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"uuid"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/db"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/domain/user/userpg"
	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/KiloProjects/kilonova/internal/auth"
	"github.com/KiloProjects/kilonova/net/discord"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer"
	"github.com/Yiling-J/theine-go"
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
}

type BaseAPI struct {
	pgx    *postgres.DB
	db     *db.DB
	mailer kilonova.Mailer
	rd     *mdrenderer.Renderer

	userRepo *userpg.Repository

	mgr *datastore.Manager

	sessionUserCache *theine.LoadingCache[string, *kilonova.UserFull]

	grader  Grader
	langMgr eval.LanguageManager

	logChan chan *logEntry

	discord discord.Provider

	evictionLogger        *slog.Logger
	testBucket            datastore.Bucket
	attachmentCacheBucket datastore.Bucket
	subtestBucket         datastore.Bucket
	avatarBucket          datastore.Bucket

	oidcProvider *op.Provider

	flagset *config.Flagset
}

// Flagset is this instance's runtime flag values. Request and job contexts carry it
// through config.WithFlagset.
func (s *BaseAPI) Flagset() *config.Flagset {
	return s.flagset
}

func (s *BaseAPI) Start(ctx context.Context) {
	if err := s.discord.Open(); err != nil {
		slog.WarnContext(ctx, "Could not connect to Discord", slog.Any("err", err))
	}
	go s.ingestAuditLogs(ctx)
	go s.cleanupBucketsJob(ctx, 30*time.Minute)
	go s.refreshProblemStatsJob(ctx, 5*time.Minute)
	go s.refreshHotProblemsJob(ctx, 4*time.Hour)
}

// PingDB reports whether the database is reachable. Used by the health endpoint,
// which must not reach past sudoapi into the pool itself.
func (s *BaseAPI) PingDB(ctx context.Context) error {
	return s.pgx.Pool().Ping(ctx)
}

func (s *BaseAPI) Close() error {
	if err := s.pgx.Close(); err != nil {
		return fmt.Errorf("couldn't close DB: %w", err)
	}

	if err := s.discord.Close(); err != nil {
		return fmt.Errorf("couldn't close Discord session: %w", err)
	}

	return nil
}

func GetBaseAPI(ctx context.Context, pgx *postgres.DB, mgr *datastore.Manager, mailer kilonova.Mailer, dc discord.Provider, flagset *config.Flagset) (*BaseAPI, error) {
	base := &BaseAPI{
		pgx:     pgx,
		db:      db.NewPSQL(pgx),
		mailer:  mailer,
		rd:      mdrenderer.NewRenderer(),
		discord: dc,
		flagset: flagset,

		mgr: mgr,

		userRepo: userpg.NewRepository(pgx),

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

	if err := base.ensureOIDCKeys(ctx); err != nil {
		return nil, fmt.Errorf("couldn't set up OIDC keys: %w", err)
	}
	authStorage, err := auth.NewAuthStorage(base.pgx, auth.StorageConfig{
		PrivateKey: flags.AuthRSAPrivateKey.Of(base.flagset),
		KeyID:      flags.AuthRSAPrivateKeyID.Of(base.flagset),

		DefaultLanguage: flags.DefaultLanguage,
	})
	if err != nil {
		return nil, err
	}
	provider, err := auth.GetProvider(auth.OauthProviderConfig{
		Storage:    authStorage,
		Debug:      kilonova.DebugMode(),
		HostPrefix: kilonova.HostPrefix(),

		CryptoKey: []byte(flags.AuthCryptoKey.Of(base.flagset)),
	})
	if err != nil {
		return nil, err
	}
	base.oidcProvider = provider
	return base, nil
}

// ensureOIDCKeys generates and persists any missing OIDC secret.
func (s *BaseAPI) ensureOIDCKeys(ctx context.Context) error {
	ctx = config.WithFlagset(ctx, s.flagset)
	if flags.AuthRSAPrivateKey.Of(s.flagset) == "" {
		key, err := auth.GeneratePrivateKey()
		if err != nil {
			return fmt.Errorf("couldn't generate RSA key: %w", err)
		}
		if err := flags.AuthRSAPrivateKey.Update(ctx, key); err != nil {
			return err
		}
	}
	if flags.AuthRSAPrivateKeyID.Of(s.flagset) == "" {
		if err := flags.AuthRSAPrivateKeyID.Update(ctx, uuid.NewV7().String()); err != nil {
			return err
		}
	}
	if flags.AuthCryptoKey.Of(s.flagset) == "" {
		if err := flags.AuthCryptoKey.Update(ctx, rand.Text()); err != nil {
			return err
		}
	}
	return nil
}

func InitQueryCounter(ctx context.Context) context.Context {
	return postgres.InitContextCounter(ctx)
}

func GetQueryCounter(ctx context.Context) int64 {
	return postgres.GetContextQueryCount(ctx)
}
