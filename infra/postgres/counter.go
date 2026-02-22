package postgres

import (
	"context"
	"sync/atomic"
)

type dbCtx string

const (
	queryCount = dbCtx("queryCount")
)

func InitContextCounter(rootCtx context.Context) context.Context {
	return context.WithValue(rootCtx, queryCount, &atomic.Int64{})
}

func GetContextQueryCount(ctx context.Context) int64 {
	cnt, ok := ctx.Value(queryCount).(*atomic.Int64)
	if !ok {
		return -1
	}
	return cnt.Load()
}
