package slicealg

import (
	"context"
	"log/slog"
)

func Map[T1, T2 any](list []T1, fn func(T1) T2) []T2 {
	if list == nil {
		return nil
	}
	result := make([]T2, len(list))
	for i := range list {
		result[i] = fn(list[i])
	}
	return result
}

func MapCtx[T1, T2 any](ctx context.Context, list []T1, fn func(context.Context, T1) (T2, error)) []T2 {
	if list == nil {
		return nil
	}
	result := make([]T2, len(list))
	for i := range list {
		var err error
		result[i], err = fn(ctx, list[i])
		if err != nil {
			slog.WarnContext(ctx, "Error running mapper", slog.Any("err", err))
		}
	}
	return result
}
