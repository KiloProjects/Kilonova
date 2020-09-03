package cache

import (
	"context"

	"github.com/go-redis/redis/v8"
)

type Cache struct {
	client *redis.Client
}

func New() (*Cache, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       0,
	})
	_, err := rdb.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}
	return &Cache{rdb}, nil
}

func (c *Cache) CreateSession(id uint) (string, error) {
	return "", nil
}

func (c *Cache) GetSession(sess string) (uint, error) {
	return 0, nil
}
