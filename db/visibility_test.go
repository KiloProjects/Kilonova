package db

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestVisibilityFunctions needs a Postgres server: set KN_TEST_DSN to run it.
// It works inside a schema of its own, which it drops when done.
func TestVisibilityFunctions(t *testing.T) {
	dsn := os.Getenv("KN_TEST_DSN")
	if dsn == "" {
		t.Skip("KN_TEST_DSN not set")
	}
	ctx := context.Background()
	conf, err := pgxpool.ParseConfig(dsn)
	require.NoError(t, err)
	// public stays on the path for the unaccent extension
	conf.ConnConfig.RuntimeParams["search_path"] = "kn_vis_test,public"
	pool, err := pgxpool.NewWithConfig(ctx, conf)
	require.NoError(t, err)
	defer pool.Close()
	_, err = pool.Exec(ctx, "DROP SCHEMA IF EXISTS kn_vis_test CASCADE; CREATE SCHEMA kn_vis_test")
	require.NoError(t, err)
	defer pool.Exec(ctx, "DROP SCHEMA kn_vis_test CASCADE")

	for _, m := range append(Migrations.Migrations, Migrations.SpecialMigrations...) {
		require.NoError(t, pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error { return m.Handler(ctx, tx) }), m.Name)
	}

	const (
		alice, bob, carol = 1, 2, 3 // unrelated user, contestant, non-contest submitter

		pbVirtualEnded, pbOfficialEnded, pbRunning, pbPublic         = 1, 2, 3, 4
		subOtherNoContest, subOwnContest, subOwnNoContest, subPublic = 1, 2, 3, 4
	)
	_, err = pool.Exec(ctx, `
INSERT INTO users (id, name, email, password) VALUES (1, 'alice', 'a@x', ''), (2, 'bob', 'b@x', ''), (3, 'carol', 'c@x', '');
INSERT INTO problems (id, name, test_name, visible) VALUES (1, 'virtual ended', 'p1', false), (2, 'official ended', 'p2', false), (3, 'running', 'p3', false), (4, 'public', 'p4', true);
INSERT INTO contests (id, name, start_time, end_time, type, visible) VALUES
	(1, 'virtual ended', NOW() - interval '2 days', NOW() - interval '1 day', 'virtual', true),
	(2, 'official ended', NOW() - interval '2 days', NOW() - interval '1 day', 'official', true),
	(3, 'private running', NOW() - interval '1 hour', NOW() + interval '1 hour', 'official', false);
INSERT INTO contest_problems (contest_id, problem_id) VALUES (1, 1), (2, 2), (3, 3);
INSERT INTO contest_registrations (user_id, contest_id) VALUES (2, 3);
INSERT INTO submissions (id, user_id, problem_id, contest_id, language) VALUES
	(1, 3, 3, NULL, 'cpp'), (2, 2, 3, 3, 'cpp'), (3, 2, 3, NULL, 'cpp'), (4, 3, 4, NULL, 'cpp');`)
	require.NoError(t, err)

	ids := func(query string, args ...any) []int {
		rows, _ := pool.Query(ctx, query, args...)
		got, err := pgx.CollectRows(rows, pgx.RowTo[int])
		require.NoError(t, err)
		return got
	}

	for _, uid := range []int{0, alice, bob, carol} {
		visible := ids("SELECT problem_id FROM visible_pbs($1)", uid)
		full := ids("SELECT problem_id FROM persistently_visible_pbs($1)", uid)
		require.Subset(t, visible, full, "fully visible must be a subset of visible for user %d", uid)
		require.NotContains(t, full, pbVirtualEnded, "ended virtual contest must not publish its problems (user %d)", uid)
		require.Contains(t, full, pbOfficialEnded)
		require.Contains(t, full, pbPublic)
	}
	require.Contains(t, ids("SELECT problem_id FROM visible_pbs($1)", bob), pbRunning)
	require.NotContains(t, ids("SELECT problem_id FROM persistently_visible_pbs($1)", bob), pbRunning)

	// A contestant sees their own submissions on the contest problem, never someone else's non-contest ones.
	for name, query := range map[string]string{
		"visible_submissions":                "SELECT sub_id FROM visible_submissions($1)",
		"visible_submissions_ex":             "SELECT sub_id FROM visible_submissions_ex($1, NULL, NULL)",
		"visible_submissions_ex, by problem": "SELECT sub_id FROM visible_submissions_ex($1, 3, NULL)",
	} {
		got := ids(query, bob)
		require.NotContains(t, got, subOtherNoContest, name)
		require.Contains(t, got, subOwnContest, name)
		require.Contains(t, got, subOwnNoContest, name)
	}
	// Non-contest submissions on a public problem stay listed for everyone.
	for _, uid := range []int{0, alice, bob} {
		require.Contains(t, ids("SELECT sub_id FROM visible_submissions($1)", uid), subPublic)
		require.Contains(t, ids("SELECT sub_id FROM visible_submissions_ex($1, NULL, NULL)", uid), subPublic)
	}
}
