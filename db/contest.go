package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
)

type dbContest struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	Name      string    `db:"name"`

	PublicJoin  bool      `db:"public_join"`
	Visible     bool      `db:"visible"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	MaxSubCount int       `db:"max_sub_count"`

	Virtual bool `db:"virtual"`

	PerUserTime           int  `db:"per_user_time"`
	RegisterDuringContest bool `db:"register_during_contest"`
}

const createContestQuery = `INSERT INTO contests (
		name, start_time, end_time
	) VALUES (
		?, ?, ?
	) RETURNING id`

func (s *DB) CreateContest(ctx context.Context, name string) (int, error) {
	if name == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	// Default is one week from now, 2 hours
	defaultStart := time.Now().AddDate(0, 0, 7).Truncate(time.Hour)
	err := s.conn.GetContext(
		ctx, &id, s.conn.Rebind(createContestQuery),
		name,
		defaultStart, defaultStart.Add(2*time.Hour),
	)
	return id, err
}

func (s *DB) Contest(ctx context.Context, id int) (*kilonova.Contest, error) {
	var contest dbContest
	err := s.conn.GetContext(ctx, &contest, "SELECT * FROM contests WHERE id = $1", id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return s.internalToContest(ctx, &contest)
}

// TODO: Test
func (s *DB) RunningContestsByProblem(ctx context.Context, problemID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := s.conn.SelectContext(
		ctx,
		&contests,
		"SELECT contests.* FROM running_contests contests, contest_problems pbs WHERE contests.id = pbs.contest_id AND pbs.problem_id = $1 ORDER BY contests.start_time DESC",
		problemID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	} else if err != nil {
		return []*kilonova.Contest{}, err
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := s.conn.SelectContext(
		ctx,
		&contests,
		"SELECT contests.* FROM contests, contest_visibility viz WHERE contests.id = viz.contest_id AND viz.user_id = $1 ORDER BY contests.start_time DESC",
		userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleFutureContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := s.conn.SelectContext(
		ctx,
		&contests,
		`SELECT contests.* FROM contests, contest_visibility viz WHERE contests.id = viz.contest_id 
		AND NOW() < contests.start_time 
		AND viz.user_id = $1 ORDER BY contests.start_time DESC`,
		userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) VisibleRunningContests(ctx context.Context, userID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := s.conn.SelectContext(
		ctx,
		&contests,
		`SELECT contests.* FROM contests, contest_visibility viz WHERE contests.id = viz.contest_id 
		AND contests.start_time <= NOW() AND NOW() < contests.end_time
		AND viz.user_id = $1 ORDER BY contests.start_time DESC`,
		userID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Contest{}, nil
	}
	return mapperCtx(ctx, contests, s.internalToContest), err
}

func (s *DB) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) error {
	toUpd, args := contestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE contests SET %s WHERE id = ?", strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) DeleteContest(ctx context.Context, id int) error {
	_, err := s.conn.ExecContext(ctx, "DELETE FROM contests WHERE id = $1", id)
	return err
}

// Contest leaderboard

type databaseTopEntry struct {
	UserID    int `db:"user_id"`
	ContestID int `db:"contest_id"`
	Total     int `db:"total_score"`
}

func (s *DB) internalToLeaderboardEntry(ctx context.Context, entry *databaseTopEntry) (*kilonova.LeaderboardEntry, error) {
	user, err := s.User(ctx, entry.UserID)
	if err != nil {
		return nil, err
	}

	var pbs = []struct {
		ProblemID int `db:"problem_id"`
		Score     int `db:"score"`
	}{}
	if err := s.conn.SelectContext(ctx, &pbs, "SELECT problem_id, score FROM max_score_contest_view WHERE user_id = $1 AND contest_id = $2", entry.UserID, entry.ContestID); err != nil {
		return nil, err
	}

	scores := make(map[int]int)
	for _, pb := range pbs {
		scores[pb.ProblemID] = pb.Score
	}

	return &kilonova.LeaderboardEntry{
		User:          user.ToBrief(),
		TotalScore:    entry.Total,
		ProblemScores: scores,
	}, nil
}

func (s *DB) ContestLeaderboard(ctx context.Context, contestID int) (*kilonova.ContestLeaderboard, error) {
	pbs, err := s.ContestProblems(ctx, contestID)
	if err != nil {
		return nil, err
	}

	leaderboard := &kilonova.ContestLeaderboard{
		ProblemOrder: mapper(pbs, func(pb *kilonova.Problem) int { return pb.ID }),
		ProblemNames: make(map[int]string),
	}
	for _, pb := range pbs {
		leaderboard.ProblemNames[pb.ID] = pb.Name
	}

	var topList []*databaseTopEntry

	err = s.conn.SelectContext(ctx, &topList, "SELECT * FROM contest_top_view WHERE contest_id = $1", contestID)
	if err != nil {
		return nil, err
	}

	leaderboard.Entries = mapperCtx(ctx, topList, s.internalToLeaderboardEntry)

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

func contestUpdateQuery(upd *kilonova.ContestUpdate) ([]string, []any) {
	toUpd, args := []string{}, []any{}
	if v := upd.Name; v != nil {
		toUpd, args = append(toUpd, "name = ?"), append(args, v)
	}
	if v := upd.PublicJoin; v != nil {
		toUpd, args = append(toUpd, "public_join = ?"), append(args, v)
	}
	if v := upd.Visible; v != nil {
		toUpd, args = append(toUpd, "visible = ?"), append(args, v)
	}
	if v := upd.StartTime; v != nil {
		toUpd, args = append(toUpd, "start_time = ?"), append(args, v)
	}
	if v := upd.EndTime; v != nil {
		toUpd, args = append(toUpd, "end_time = ?"), append(args, v)
	}
	if v := upd.MaxSubs; v != nil {
		toUpd, args = append(toUpd, "max_sub_count = ?"), append(args, v)
	}
	if v := upd.PerUserTime; v != nil {
		toUpd, args = append(toUpd, "per_user_time = ?"), append(args, v)
	}
	if v := upd.RegisterDuringContest; v != nil {
		toUpd, args = append(toUpd, "register_during_contest = ?"), append(args, v)
	}

	return toUpd, args
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

		PerUserTime: contest.PerUserTime,

		RegisterDuringContest: contest.RegisterDuringContest,

		Visible: contest.Visible,
	}, nil
}
