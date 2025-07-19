package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	sq "github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
)

type dbContest struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Name      string    `db:"name"`
	Desc      string    `db:"description"`

	PublicJoin  bool      `db:"public_join"`
	Visible     bool      `db:"visible"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	MaxSubCount int       `db:"max_sub_count"`

	Virtual bool `db:"virtual"`

	PublicLeaderboard bool `db:"public_leaderboard"`

	LeaderboardStyle      kilonova.LeaderboardType `db:"leaderboard_style"`
	LeaderboardFreezeTime *time.Time               `db:"leaderboard_freeze_time"`
	LeaderboardAdvFilter  bool                     `db:"leaderboard_advanced_filter"`

	ICPCSubmissionPenalty int `db:"icpc_submission_penalty"`

	PerUserTime           int  `db:"per_user_time"`
	RegisterDuringContest bool `db:"register_during_contest"`

	SubmissionCooldown int `db:"submission_cooldown_ms"`
	QuestionCooldown   int `db:"question_cooldown_ms"`

	Type kilonova.ContestType `db:"type"`
}

const createContestQuery = `INSERT INTO contests (
		name, type, start_time, end_time
	) VALUES (
		$1, $2, $3, $4
	) RETURNING id`

func (s *DB) CreateContest(ctx context.Context, name string, cType kilonova.ContestType) (int, error) {
	if name == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	// Default is one week from now, 2 hours
	defaultStart := time.Now().AddDate(0, 0, 7).Truncate(time.Hour)
	err := s.conn.QueryRow(
		ctx, createContestQuery,
		name, cType,
		defaultStart, defaultStart.Add(2*time.Hour),
	).Scan(&id)
	return id, err
}

// TODO: Convert to taking Contests() and returning the first one
func (s *DB) Contest(ctx context.Context, id int) (*kilonova.Contest, error) {
	var contest dbContest
	err := Get(s.conn, ctx, &contest, "SELECT * FROM contests WHERE id = $1", id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return s.internalToContest(ctx, &contest)
}

func (s *DB) Contests(ctx context.Context, filter kilonova.ContestFilter) ([]*kilonova.Contest, error) {
	fb := newFilterBuilder()
	contestFilterQuery(&filter, fb)

	rows, _ := s.conn.Query(ctx, fmt.Sprintf(
		"SELECT * FROM contests WHERE %s %s %s",
		fb.Where(), getContestOrdering(filter.Ordering, filter.Ascending), FormatLimitOffset(filter.Limit, filter.Offset),
	), fb.Args()...)
	contests, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[dbContest])
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	} else if err != nil {
		return []*kilonova.Contest{}, err
	}

	return mapperCtx(ctx, contests, s.internalToContest), nil
}

func (s *DB) ContestCount(ctx context.Context, filter kilonova.ContestFilter) (int, error) {
	fb := newFilterBuilder()
	contestFilterQuery(&filter, fb)
	var val int
	err := s.conn.QueryRow(ctx, "SELECT COUNT(*) FROM contests WHERE "+fb.Where(), fb.Args()...).Scan(&val)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return -1, err
		}
		val = 0
	}
	return val, nil
}

func contestFilterQuery(filter *kilonova.ContestFilter, fb *filterBuilder) {
	if v := filter.ID; v != nil {
		fb.AddConstraint("id = %s", v)
	}
	if v := filter.IDs; v != nil {
		fb.AddConstraint("id = ANY(%s)", v)
	}
	if filter.Look {
		var id int
		if filter.LookingUser != nil {
			id = filter.LookingUser.ID
		}
		fb.AddConstraint("EXISTS (SELECT 1 FROM visible_contests(%s) viz WHERE contests.id = viz.contest_id)", id)
	}

	if v := filter.ProblemID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM contest_problems pbs WHERE contests.id = pbs.contest_id AND pbs.problem_id = %s)", v)
	}
	if v := filter.ContestantID; v != nil {
		fb.AddConstraint("EXISTS (SELECT 1 FROM contest_registrations regs WHERE contests.id = regs.contest_id AND regs.user_id = %s)", v)
	}
	if v := filter.EditorID; v != nil {
		if filter.NotEditor {
			fb.AddConstraint("NOT EXISTS (SELECT 1 FROM contest_user_access acc WHERE contests.id = acc.contest_id AND acc.user_id = %s AND acc.access = 'editor')", v)
		} else {
			fb.AddConstraint("EXISTS (SELECT 1 FROM contest_user_access acc WHERE contests.id = acc.contest_id AND acc.user_id = %s AND acc.access = 'editor')", v)
		}
	}

	if filter.Future {
		fb.AddConstraint("NOW() < start_time")
	}
	if filter.Running {
		fb.AddConstraint("start_time <= NOW()")
		fb.AddConstraint("NOW() < end_time")
	}
	if filter.Ended {
		fb.AddConstraint("end_time <= NOW()")
	}

	if v := filter.Type; v != kilonova.ContestTypeNone {
		fb.AddConstraint("type = %s", v)
	}

	if v := filter.Since; v != nil {
		fb.AddConstraint("created_at > %s", v)
	}

	// See field comment for details
	if v := filter.ImportantContestsUID; v != nil {
		fb.AddConstraint(`(
				(type = 'official' AND end_time >= NOW()) 
				OR 
				EXISTS (SELECT 1 FROM contest_registrations regs WHERE contests.id = regs.contest_id AND regs.user_id = %s)
				OR
				EXISTS (SELECT 1 FROM contest_user_access acc WHERE contests.id = acc.contest_id AND acc.user_id = %s)
			)`, v, v)
	}

}

func (s *DB) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) error {
	ub := newUpdateBuilder()
	contestUpdateQuery(&upd, ub)
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)
	_, err := s.conn.Exec(ctx, "UPDATE contests SET "+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteContest(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM contests WHERE id = $1", id)
	return err
}

// Contest leaderboard

type databaseClassicEntry struct {
	UserID    int             `db:"user_id"`
	ContestID int             `db:"contest_id"`
	Total     decimal.Decimal `db:"total_score"`
	LastTime  *time.Time      `db:"last_time"`

	FreezeTime *time.Time `db:"freeze_time"`
}

func (s *DB) classicToLeaderboardEntry(ctx context.Context, entry *databaseClassicEntry) (*kilonova.LeaderboardEntry, error) {
	user, err := s.User(ctx, kilonova.UserFilter{ID: &entry.UserID})
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, "SELECT problem_id, score FROM contest_max_scores($2, $3) WHERE user_id = $1", entry.UserID, entry.ContestID, entry.FreezeTime)
	pbs, err := pgx.CollectRows(rows, pgx.RowToStructByName[struct {
		ProblemID int             `db:"problem_id"`
		Score     decimal.Decimal `db:"score"`
	}])
	if err != nil {
		return nil, err
	}

	var numSolved int
	scores := make(map[int]decimal.Decimal)
	for _, pb := range pbs {
		scores[pb.ProblemID] = pb.Score
		if pb.Score.Equal(decimal.NewFromInt(100)) {
			numSolved++
		}
	}

	return &kilonova.LeaderboardEntry{
		User:          user.Brief(),
		TotalScore:    entry.Total,
		ProblemScores: scores,

		ProblemAttempts: make(map[int]int),
		Penalty:         0,
		NumSolved:       numSolved,
		ProblemTimes:    make(map[int]float64),

		LastTime:   entry.LastTime,
		FreezeTime: entry.FreezeTime,
	}, nil
}

func (s *DB) contestMaxScores(contestID int, freezeTime *time.Time, userID *int, problemID *int) sq.SelectBuilder {
	maxSubmissionStrat := sq.Select(
		"user_id",
		"problem_id",
		"FIRST_VALUE(score * (leaderboard_score_scale / 100)) OVER w AS max_score",
		"FIRST_VALUE(created_at) OVER w AS mintime").Distinct().
		From("submissions").Where("contest_id = ?", contestID).
		Where("created_at <= COALESCE(?, NOW())", freezeTime).
		Where("(status = 'finished' OR status = 'reevaling')").
		Suffix("WINDOW w AS (PARTITION BY user_id, problem_id ORDER BY score DESC, created_at ASC)")

	subtaskMaxScores := sq.Select(
		"user_id",
		"subtask_id",
		"problem_id",
		"FIRST_VALUE(computed_score * (leaderboard_score_scale / 100)) OVER w AS max_score",
		"FIRST_VALUE(created_at) OVER w AS mintime").Distinct().
		From("submission_subtasks").Where("contest_id = ?", contestID).Where("subtask_id IS NOT NULL").
		Suffix("WINDOW w AS (PARTITION BY user_id, subtask_id, problem_id ORDER BY computed_score DESC, created_at ASC)")

	sumSubtasksStrat := sq.Select("user_id", "problem_id", "coalesce(SUM(max_score), -1) AS max_score", "MAX(mintime) AS mintime").
		FromSelect(subtaskMaxScores, "max_scores").
		GroupBy("user_id", "problem_id")

	if userID != nil {
		maxSubmissionStrat = maxSubmissionStrat.Where("user_id = ?", userID)
	}
	if problemID != nil {
		maxSubmissionStrat = maxSubmissionStrat.Where("problem_id = ?", problemID)
	}

	maxScores := sq.Select("contest_users.user_id AS user_id", "pbs.problem_id AS problem_id")
	maxScores = maxScores.Column(sq.Alias(sq.Case().When(
		sq.Expr("problems.scoring_strategy IN ('max_submission', 'acm-icpc')"),
		"COALESCE(ms_sub.max_score, -1)",
	).When(
		sq.Expr("problems.scoring_strategy = 'sum_subtasks'"),
		"COALESCE(ms_subtask.max_score, -1)",
	).Else("-1"), "score",
	))
	maxScores = maxScores.Column(sq.Alias(sq.Case().When(
		sq.Expr("problems.scoring_strategy IN ('max_submission', 'acm-icpc')"),
		"COALESCE(ms_sub.mintime, NULL)",
	).When(
		sq.Expr("problems.scoring_strategy = 'sum_subtasks'"),
		"COALESCE(ms_subtask.mintime, NULL)",
	).Else("NULL"), "mintime",
	)).From("contest_problems pbs").
		InnerJoin("contest_registrations contest_users ON contest_users.contest_id = pbs.contest_id").
		InnerJoin("problems ON pbs.problem_id = problems.id").
		JoinClause(maxSubmissionStrat.Prefix("LEFT JOIN (").Suffix(") AS ms_sub ON (ms_sub.user_id = contest_users.user_id AND ms_sub.problem_id = pbs.problem_id)")).
		JoinClause(sumSubtasksStrat.Prefix("LEFT JOIN (").Suffix(") AS ms_subtask ON (ms_subtask.user_id = contest_users.user_id AND ms_subtask.problem_id = pbs.problem_id)")).
		Where(sq.Eq{"pbs.contest_id": contestID})

	if userID != nil {
		maxScores = maxScores.Where("contest_users.user_id = ?", userID)
	}
	if problemID != nil {
		maxScores = maxScores.Where("pbs.problem_id = ?", problemID)
	}

	return maxScores

}

func (s *DB) contestTopView(contestID int, freezeTime *time.Time, includeEditors bool) sq.SelectBuilder {
	contestScores := sq.Select("user_id", "SUM(score) AS total_score", "MAX(mintime) FILTER (WHERE score > 0) AS last_time").
		FromSelect(s.contestMaxScores(contestID, freezeTime, nil, nil), "contest_max_scores").
		Where("score > 0").
		GroupBy("user_id")

	legitContestants := sq.Select("regs.*").From("contest_registrations regs").Where("regs.contest_id = ?", contestID)
	if !includeEditors {
		legitContestants = legitContestants.Where("NOT EXISTS (SELECT 1 FROM contest_user_access acc WHERE acc.user_id = regs.user_id AND acc.contest_id = regs.contest_id)")
	}

	return sq.Select().
		Column("contest_users.user_id").
		Column("?::integer AS contest_id", contestID).
		Column("COALESCE(scores.total_score, 0) AS total_score").
		Column("last_time").
		FromSelect(legitContestants, "contest_users").
		JoinClause(contestScores.Prefix("LEFT JOIN (").Suffix(") AS scores ON contest_users.user_id = scores.user_id")).
		OrderBy("total_score DESC", "last_time ASC NULLS LAST")
}

// TODO
// func (s *DB) contestICPCView(contestID int, freezeTime *time.Time, includeEditors bool) sq.SelectBuilder {
// 	legitContestants := sq.Select("regs.*").From("contest_registrations regs").Where("regs.contest_id = ?", contestID)
// 	if !includeEditors {
// 		legitContestants = legitContestants.Where("NOT EXISTS (SELECT 1 FROM contest_user_access acc WHERE acc.user_id = regs.user_id AND acc.contest_id = regs.contest_id)")
// 	}

// 	solvedPbs := sq.Select("user_id", "problem_id", "mintime AS last_time").
// 		FromSelect(s.contestMaxScores(contestID, freezeTime, nil, nil), "contest_max_scores").
// 		Where("score = 100")

// 	lastTimes := sq.Select("user_id", "MAX(last_time) AS last_time", "COUNT(*) AS num_problems", "MAX_").
// 		FromSelect(solvedPbs, "solved_pbs").GroupBy("user_id")
// }

func (s *DB) ContestClassicLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time, generated *bool) (*kilonova.ContestLeaderboard, error) {
	pbs, err := s.ContestProblems(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	leaderboard := &kilonova.ContestLeaderboard{
		ProblemOrder: mapper(pbs, func(pb *kilonova.Problem) int { return pb.ID }),
		ProblemNames: make(map[int]string),

		AdvancedFilter: contest.LeaderboardAdvancedFilter,

		FreezeTime: freezeTime,
		Type:       kilonova.LeaderboardTypeClassic,
	}
	for _, pb := range pbs {
		leaderboard.ProblemNames[pb.ID] = pb.Name
	}

	sb := sq.Select("*").Column("?::timestamptz AS freeze_time", freezeTime).
		FromSelect(s.contestTopView(contest.ID, freezeTime, contest.Type == kilonova.ContestTypeVirtual), "contest_top_view").
		OrderBy("total_score DESC", "last_time ASC NULLS LAST", "user_id")

	if generated != nil {
		sb = sb.Where("EXISTS (SELECT 1 FROM users WHERE user_id = users.id AND generated = ?)", *generated)
	}

	query, args, err := sb.ToSql()
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, query, args...)
	topList, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[databaseClassicEntry])
	if err != nil {
		return nil, err
	}

	leaderboard.Entries = mapperCtx(ctx, topList, s.classicToLeaderboardEntry)

	return leaderboard, nil
}

// TODO: Update
type databaseICPCEntry struct {
	UserID    int        `db:"user_id"`
	ContestID int        `db:"contest_id"`
	LastTime  *time.Time `db:"last_time"`
	NumSolved int        `db:"num_solved"`
	Penalty   int        `db:"penalty"`

	NumAttempts int `db:"num_attempts"`

	FreezeTime *time.Time `db:"freeze_time"`
}

func (s *DB) icpcToLeaderboardEntry(ctx context.Context, entry *databaseICPCEntry) (*kilonova.LeaderboardEntry, error) {
	user, err := s.User(ctx, kilonova.UserFilter{ID: &entry.UserID})
	if err != nil {
		return nil, err
	}

	rows, _ := s.conn.Query(ctx, `
		SELECT problem_id, score, mintime, COALESCE(natts.num_atts, 0) AS num_attempts
			FROM contest_max_scores($2, $3) cms, 
			LATERAL (SELECT COUNT(*) AS num_atts FROM submissions 
				WHERE contest_id = $2 AND user_id = $1 AND compile_error = false AND (status = 'finished' OR status = 'reevaling') AND problem_id = cms.problem_id AND created_at <= COALESCE($3, NOW()) AND (cms.score < 100 OR created_at < cms.mintime)) natts 
		WHERE user_id = $1
`, entry.UserID, entry.ContestID, entry.FreezeTime)
	pbs, err := pgx.CollectRows(rows, pgx.RowToStructByName[struct {
		ProblemID int             `db:"problem_id"`
		Score     decimal.Decimal `db:"score"`
		MinTime   *time.Time      `db:"mintime"`
		Attempts  int             `db:"num_attempts"`
	}])
	if err != nil {
		return nil, err
	}

	scores := make(map[int]decimal.Decimal)
	times := make(map[int]float64)
	attempts := make(map[int]int)
	for _, pb := range pbs {
		scores[pb.ProblemID] = pb.Score
		if pb.MinTime != nil && util.ContestContext(ctx) != nil {
			times[pb.ProblemID] = pb.MinTime.Sub(util.ContestContext(ctx).StartTime).Minutes()
		}
		attempts[pb.ProblemID] = pb.Attempts
	}

	return &kilonova.LeaderboardEntry{
		User:          user.Brief(),
		TotalScore:    decimal.Zero,
		ProblemScores: scores,

		ProblemAttempts: attempts,
		ProblemTimes:    times,
		Penalty:         entry.Penalty,
		NumSolved:       entry.NumSolved,

		LastTime:   entry.LastTime,
		FreezeTime: entry.FreezeTime,
	}, nil
}

func (s *DB) ContestICPCLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time, generated *bool) (*kilonova.ContestLeaderboard, error) {
	pbs, err := s.ContestProblems(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	leaderboard := &kilonova.ContestLeaderboard{
		ProblemOrder: mapper(pbs, func(pb *kilonova.Problem) int { return pb.ID }),
		ProblemNames: make(map[int]string),

		AdvancedFilter: contest.LeaderboardAdvancedFilter,

		FreezeTime: freezeTime,
		Type:       kilonova.LeaderboardTypeICPC,
	}
	for _, pb := range pbs {
		leaderboard.ProblemNames[pb.ID] = pb.Name
	}

	var topList []*databaseICPCEntry

	where, args := "1 = 1", []any{contest.ID, freezeTime, contest.Type == kilonova.ContestTypeVirtual}
	if generated != nil {
		where = "EXISTS (SELECT 1 FROM users WHERE user_id = users.id AND generated = ?)"
		args = append(args, *generated)
	}

	err = Select(s.conn, ctx, &topList, "SELECT *, $2 AS freeze_time FROM contest_icpc_view($1, $2, $3) WHERE "+where+" ORDER BY num_solved DESC, penalty ASC NULLS LAST, last_time ASC NULLS LAST, user_id", args...)
	if err != nil {
		slog.WarnContext(ctx, "Couldn't get ICPC leaderboard", slog.Any("err", err))
		return nil, err
	}

	leaderboard.Entries = mapperCtx(context.WithValue(ctx, util.ContestKey, contest), topList, s.icpcToLeaderboardEntry)

	return leaderboard, nil
}

// MOSS setup

func (s *DB) InsertMossSubmission(ctx context.Context, contestID int, problemID int, lang string, subcount int) (int, error) {
	var id int
	err := s.conn.QueryRow(ctx, "INSERT INTO moss_submissions (contest_id, problem_id, language, subcount, url) VALUES ($1, $2, $3, $4, '') RETURNING id", contestID, problemID, lang, subcount).Scan(&id)
	return id, err
}

func (s *DB) SetMossURL(ctx context.Context, subID int, url string) error {
	_, err := s.conn.Exec(ctx, "UPDATE moss_submissions SET url = $1 WHERE id = $2", url, subID)
	return err
}

func (s *DB) MossSubmissions(ctx context.Context, contestID int) ([]*kilonova.MOSSSubmission, error) {
	rows, _ := s.conn.Query(ctx, "SELECT * FROM moss_submissions WHERE contest_id = $1 ORDER BY created_at DESC", contestID)
	subs, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[kilonova.MOSSSubmission])
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return subs, nil
}

// Contest problems

func (s *DB) UpdateContestProblems(ctx context.Context, contestID int, problems []int) error {
	return s.updateManyToMany(ctx, "contest_problems", "contest_id", "problem_id", contestID, problems, true)
}

// Access rights

func (s *DB) AddContestEditor(ctx context.Context, contestID int, uid int) error {
	return s.addAccess(ctx, "contest_user_access", "contest_id", contestID, uid, "editor")
}

func (s *DB) AddContestTester(ctx context.Context, contestID int, uid int) error {
	return s.addAccess(ctx, "contest_user_access", "contest_id", contestID, uid, "viewer")
}

func (s *DB) StripContestAccess(ctx context.Context, contestID int, uid int) error {
	return s.removeAccess(ctx, "contest_user_access", "contest_id", contestID, uid)
}

func (s *DB) contestEditors(ctx context.Context, contestID int) ([]*kilonova.UserFull, error) {
	return s.getAccessUsers(ctx, "contest_user_access", "contest_id", contestID, accessEditor)
}

func (s *DB) contestViewers(ctx context.Context, contestID int) ([]*kilonova.UserFull, error) {
	return s.getAccessUsers(ctx, "contest_user_access", "contest_id", contestID, accessViewer)
}

func contestUpdateQuery(upd *kilonova.ContestUpdate, ub *updateBuilder) {
	if v := upd.Name; v != nil {
		ub.AddUpdate("name = %s", v)
	}
	if v := upd.Description; v != nil {
		ub.AddUpdate("description = %s", v)
	}
	if v := upd.PublicJoin; v != nil {
		ub.AddUpdate("public_join = %s", v)
	}
	if v := upd.Visible; v != nil {
		ub.AddUpdate("visible = %s", v)
	}
	if v := upd.StartTime; v != nil {
		ub.AddUpdate("start_time = %s", v)
	}
	if v := upd.EndTime; v != nil {
		ub.AddUpdate("end_time = %s", v)
	}
	if v := upd.MaxSubs; v != nil {
		ub.AddUpdate("max_sub_count = %s", v)
	}
	if v := upd.PublicLeaderboard; v != nil {
		ub.AddUpdate("public_leaderboard = %s", v)
	}
	if v := upd.LeaderboardStyle; v != kilonova.LeaderboardTypeNone {
		ub.AddUpdate("leaderboard_style = %s", v)
	}
	if v := upd.LeaderboardFreeze; upd.ChangeLeaderboardFreeze {
		ub.AddUpdate("leaderboard_freeze_time = %s", v)
	}
	if v := upd.ICPCSubmissionPenalty; v != nil {
		ub.AddUpdate("icpc_submission_penalty = %s", v)
	}
	if v := upd.LeaderboardAdvancedFilter; v != nil {
		ub.AddUpdate("leaderboard_advanced_filter = %s", v)
	}
	if v := upd.QuestionCooldown; v != nil {
		ub.AddUpdate("question_cooldown_ms = %s", v)
	}
	if v := upd.SubmissionCooldown; v != nil {
		ub.AddUpdate("submission_cooldown_ms = %s", v)
	}
	if v := upd.PerUserTime; v != nil {
		ub.AddUpdate("per_user_time = %s", v)
	}
	if v := upd.RegisterDuringContest; v != nil {
		ub.AddUpdate("register_during_contest = %s", v)
	}
	if v := upd.Type; v != kilonova.ContestTypeNone {
		ub.AddUpdate("type = %s", v)
	}
}

func getContestOrdering(ordering string, ascending bool) string {
	ord := " DESC"
	if ascending {
		ord = " ASC"
	}
	switch ordering {
	case "id":
		return "ORDER BY id" + ord
	case "end_time":
		return "ORDER BY end_time" + ord + ", id ASC"
	default: // case "start_time":
		return "ORDER BY start_time" + ord + ", id ASC"
	}
}

func (s *DB) internalToContest(ctx context.Context, contest *dbContest) (*kilonova.Contest, error) {

	editors, err := s.contestEditors(ctx, contest.ID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get contest editors", slog.Any("err", err))
		editors = []*kilonova.UserFull{}
	}

	viewers, err := s.contestViewers(ctx, contest.ID)
	if err != nil {
		slog.WarnContext(ctx, "Could not get contest viewers", slog.Any("err", err))
		viewers = []*kilonova.UserFull{}
	}

	return &kilonova.Contest{
		ID:         contest.ID,
		CreatedAt:  contest.CreatedAt,
		Name:       contest.Name,
		Editors:    mapper(editors, (*kilonova.UserFull).Brief),
		Testers:    mapper(viewers, (*kilonova.UserFull).Brief),
		PublicJoin: contest.PublicJoin,
		StartTime:  contest.StartTime,
		EndTime:    contest.EndTime,
		MaxSubs:    contest.MaxSubCount,

		Description: contest.Desc,

		PerUserTime: contest.PerUserTime,

		PublicLeaderboard: contest.PublicLeaderboard,
		LeaderboardStyle:  contest.LeaderboardStyle,
		LeaderboardFreeze: contest.LeaderboardFreezeTime,

		LeaderboardAdvancedFilter: contest.LeaderboardAdvFilter,

		ICPCSubmissionPenalty: contest.ICPCSubmissionPenalty,

		RegisterDuringContest: contest.RegisterDuringContest,

		SubmissionCooldown: time.Millisecond * time.Duration(contest.SubmissionCooldown),
		QuestionCooldown:   time.Millisecond * time.Duration(contest.QuestionCooldown),

		Visible: contest.Visible,
		Type:    contest.Type,
	}, nil
}
