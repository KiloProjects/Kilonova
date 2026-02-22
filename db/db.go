package db

import (
	"context"
	"fmt"

	"github.com/KiloProjects/kilonova/infra/postgres"
	"github.com/KiloProjects/kilonova/internal/repository"
	"github.com/KiloProjects/kilonova/util/slicealg"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	conn     *pgxpool.Pool
	userRepo *repository.UserRepository
}

type Queryer interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Deprecated: It's just a placeholder for old behavior
func Get[T any](pgconn Queryer, ctx context.Context, dest *T, query string, args ...any) error {
	rows, _ := pgconn.Query(ctx, query, args...)
	val, err := pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[T])
	if err != nil {
		return err
	}
	*dest = val
	return nil
}

// Deprecated: It's just a placeholder for old behavior
func Select[T any](pgconn Queryer, ctx context.Context, dest *[]*T, query string, args ...any) error {
	rows, _ := pgconn.Query(ctx, query, args...)
	vals, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByNameLax[T])
	if err != nil {
		return err
	}
	*dest = vals
	return nil
}

func NewPSQL(conn *postgres.DB) *DB {
	return &DB{conn.Pool(), repository.NewUserRepository(conn.Pool())}
}

func FormatLimitOffset(limit int, offset int) string {
	if limit > 0 && offset > 0 {
		return fmt.Sprintf("LIMIT %d OFFSET %d", limit, offset)
	}

	if limit > 0 {
		return fmt.Sprintf("LIMIT %d", limit)
	}

	if offset > 0 {
		return fmt.Sprintf("OFFSET %d", offset)
	}

	return ""
}

// Deprecated: use slicealg.Map instead
func mapper[T1 any, T2 any](lst []T1, f func(T1) T2) []T2 {
	return slicealg.Map(lst, f)
}

// Deprecated: use slicealg.MapCtx instead
func mapperCtx[T1 any, T2 any](ctx context.Context, lst []T1, f func(context.Context, T1) (T2, error)) []T2 {
	return slicealg.MapCtx(ctx, lst, f)
}

func toSingular[T1, T2 any](ctx context.Context, filter T1, f func(ctx context.Context, filter T1) ([]*T2, error)) (*T2, error) {
	many, err := f(ctx, filter)
	if err != nil || len(many) == 0 {
		return nil, err
	}
	return many[0], nil
}
