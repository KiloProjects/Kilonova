package rclient

import (
	"context"
	"strconv"
	"time"

	"github.com/KiloProjects/Kilonova/internal/config"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
)

type RClient struct {
	client *redis.Client
}

func New() (*RClient, error) {
	rdb := redis.NewClient(config.C.Cache.GenOptions())
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &RClient{rdb}, nil
}

func (c *RClient) CreateSession(ctx context.Context, uid int64) (string, error) {
	id := uuid.New()
	_, err := c.client.Set(ctx, id.String(), strconv.FormatInt(uid, 10), time.Hour*24*30).Result()
	if err != nil {
		return "", err
	}
	return id.String(), nil
}

func (c *RClient) GetSession(ctx context.Context, sess string) (int64, error) {
	id, err := uuid.Parse(sess)
	if err != nil {
		return -1, err
	}
	val, err := c.client.Get(ctx, id.String()).Result()
	if err != nil {
		return -1, err
	}
	uid, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func (c *RClient) RemoveSession(ctx context.Context, sess string) error {
	id, err := uuid.Parse(sess)
	if err != nil {
		return err
	}
	_, err = c.client.Del(ctx, id.String()).Result()
	return err
}
