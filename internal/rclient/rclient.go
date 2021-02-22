package rclient

import (
	"context"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/go-redis/redis/v8"
)

type RClient struct {
	client *redis.Client
}

func New() (*RClient, error) {
	rdb := redis.NewClient(config.Cache.GenOptions())
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &RClient{rdb}, nil
}

func sessName(sess string) string {
	return "sessions:" + sess
}

func (c *RClient) CreateSession(ctx context.Context, uid int) (string, error) {
	id := kilonova.RandomString(16)
	_, err := c.client.Set(ctx, sessName(id), strconv.Itoa(uid), time.Hour*24*30).Result()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (c *RClient) GetSession(ctx context.Context, sess string) (int, error) {
	val, err := c.client.Get(ctx, sessName(sess)).Result()
	if err != nil {
		return -1, err
	}
	uid, err := strconv.Atoi(val)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func (c *RClient) RemoveSession(ctx context.Context, sess string) error {
	_, err := c.client.Del(ctx, sessName(sess)).Result()
	return err
}

func verifName(verif string) string {
	return "verification:" + verif
}

func (c *RClient) CreateVerification(ctx context.Context, uid int) (string, error) {
	id := kilonova.RandomString(16)
	_, err := c.client.Set(ctx, verifName(id), strconv.Itoa(uid), time.Hour*24*30).Result()
	if err != nil {
		return "", err
	}
	return id, nil
}

func (c *RClient) GetVerification(ctx context.Context, verif string) (int, error) {
	val, err := c.client.Get(ctx, verifName(verif)).Result()
	if err != nil {
		return -1, err
	}
	uid, err := strconv.Atoi(val)
	if err != nil {
		return -1, err
	}
	return uid, nil
}

func (c *RClient) RemoveVerification(ctx context.Context, verif string) error {
	_, err := c.client.Del(ctx, verifName(verif)).Result()
	return err
}
