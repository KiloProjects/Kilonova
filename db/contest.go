package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
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

	ICPCSubmissionPenalty int `db:"icpc_submission_penalty"`

	PerUserTime           int  `db:"per_user_time"`
	RegisterDuringContest bool `db:"register_during_contest"`
}

const createContestQuery = `INSERT INTO contests (
		name, start_time, end_time
	) VALUES (
		$1, $2, $3
	) RETURNING id`

func (s *DB) CreateContest(ctx context.Context, name string) (int, error) {
	if name == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	// Default is one week from now, 2 hours
	defaultStart := time.Now().AddDate(0, 0, 7).Truncate(time.Hour)
	err := s.conn.QueryRow(
		ctx, createContestQuery,
		name,
		defaultStart, defaultStart.Add(2*time.Hour),
	).Scan(&id)
	return id, err
}

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

// TODO: Test
// TODO: it might expose hidden running contests with that problem
func (s *DB) RunningContestsByProblem(ctx context.Context, problemID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := Select(s.conn,
		ctx,
		&contests,
		"SELECT contests.* FROM running_contests contests, contest_problems pbs WHERE contests.id = pbs.contest_id AND pbs.problem_id = $1 ORDER BY contests.start_time DESC, contests.id ASC",
		problemID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	} else if err != nil {
		return []*kilonova.Contest{}, err
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := Select(s.conn,
		ctx,
		&contests,
		"SELECT contests.* FROM contests WHERE EXISTS (SELECT 1 FROM visible_contests($1) viz WHERE contests.id = viz.contest_id) ORDER BY contests.start_time DESC, contests.id ASC",
		userID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleFutureContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := Select(s.conn,
		ctx,
		&contests,
		`SELECT contests.* FROM contests WHERE EXISTS (SELECT 1 FROM visible_contests($1) viz WHERE contests.id = viz.contest_id)
		AND NOW() < contests.start_time ORDER BY contests.start_time DESC, contests.id ASC`,
		userID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleRunningContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := Select(s.conn,
		ctx,
		&contests,
		`SELECT contests.* FROM contests WHERE EXISTS (SELECT 1 FROM visible_contests($1) viz WHERE contests.id = viz.contest_id) 
		AND contests.start_time <= NOW() AND NOW() < contests.end_time ORDER BY contests.start_time DESC, contests.id ASC`,
		userID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
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
		User:          user.ToBrief(),
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

func (s *DB) ContestClassicLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time) (*kilonova.ContestLeaderboard, error) {
	pbs, err := s.ContestProblems(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	leaderboard := &kilonova.ContestLeaderboard{
		ProblemOrder: mapper(pbs, func(pb *kilonova.Problem) int { return pb.ID }),
		ProblemNames: make(map[int]string),

		FreezeTime: freezeTime,
		Type:       kilonova.LeaderboardTypeClassic,
	}
	for _, pb := range pbs {
		leaderboard.ProblemNames[pb.ID] = pb.Name
	}

	var topList []*databaseClassicEntry

	err = Select(s.conn, ctx, &topList, "SELECT *, $2 AS freeze_time FROM contest_top_view($1, $2)", contest.ID, freezeTime)
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
				WHERE contest_id = $2 AND user_id = $1 AND status = 'finished' AND problem_id = cms.problem_id AND created_at <= COALESCE($3, NOW()) AND (cms.score < 100 OR created_at < cms.mintime)) natts 
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
		User:          user.ToBrief(),
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

func (s *DB) ContestICPCLeaderboard(ctx context.Context, contest *kilonova.Contest, freezeTime *time.Time) (*kilonova.ContestLeaderboard, error) {
	pbs, err := s.ContestProblems(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	leaderboard := &kilonova.ContestLeaderboard{
		ProblemOrder: mapper(pbs, func(pb *kilonova.Problem) int { return pb.ID }),
		ProblemNames: make(map[int]string),

		FreezeTime: freezeTime,
		Type:       kilonova.LeaderboardTypeICPC,
	}
	for _, pb := range pbs {
		leaderboard.ProblemNames[pb.ID] = pb.Name
	}

	var topList []*databaseICPCEntry

	err = Select(s.conn, ctx, &topList, "SELECT *, $2 AS freeze_time FROM contest_icpc_view($1, $2)", contest.ID, freezeTime)
	if err != nil {
		zap.S().Warn(err)
		return nil, err
	}

	leaderboard.Entries = mapperCtx(context.WithValue(ctx, util.ContestKey, contest), topList, s.icpcToLeaderboardEntry)

	return leaderboard, nil
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

func (s *DB) contestEditors(ctx context.Context, contestID int) ([]*User, error) {
	return s.getAccessUsers(ctx, "contest_user_access", "contest_id", contestID, accessEditor)
}

func (s *DB) contestViewers(ctx context.Context, contestID int) ([]*User, error) {
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
	if v := upd.PerUserTime; v != nil {
		ub.AddUpdate("per_user_time = %s", v)
	}
	if v := upd.RegisterDuringContest; v != nil {
		ub.AddUpdate("register_during_contest = %s", v)
	}
}

func (s *DB) internalToContest(ctx context.Context, contest *dbContest) (*kilonova.Contest, error) {

	editors, err := s.contestEditors(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	viewers, err := s.contestViewers(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	return &kilonova.Contest{
		ID:         contest.ID,
		CreatedAt:  contest.CreatedAt,
		Name:       contest.Name,
		Editors:    mapper(editors, toUserBrief),
		Testers:    mapper(viewers, toUserBrief),
		PublicJoin: contest.PublicJoin,
		StartTime:  contest.StartTime,
		EndTime:    contest.EndTime,
		MaxSubs:    contest.MaxSubCount,

		Description: contest.Desc,

		PerUserTime: contest.PerUserTime,

		PublicLeaderboard: contest.PublicLeaderboard,
		LeaderboardStyle:  contest.LeaderboardStyle,
		LeaderboardFreeze: contest.LeaderboardFreezeTime,

		ICPCSubmissionPenalty: contest.ICPCSubmissionPenalty,

		RegisterDuringContest: contest.RegisterDuringContest,

		Visible: contest.Visible,
	}, nil
}
