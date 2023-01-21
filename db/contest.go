package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
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
		return nil, nil
	}
	return s.internalToContest(ctx, &contest)
}

// TODO: Test
func (s *DB) ContestsByProblem(ctx context.Context, problemID int) ([]*kilonova.Contest, error) {
	var contests []*dbContest
	err := s.conn.SelectContext(
		ctx,
		&contests,
		"SELECT contests.* FROM contests, contest_problems pbs WHERE contests.id = pbs.contest_id AND pbs.problem_id = $1 ORDER BY contests.start_time DESC",
		problemID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.Contest{}, nil
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

// func (s *DB) Contests(ctx context.Context, filter kilonova.ContestFilter) ([]*kilonova.Contest, error) {
// 	var contests []*dbContest
// 	where, args := contestFilterQuery(&filter)
// 	query := s.conn.Rebind("SELECT * FROM contests WHERE " + strings.Join(where, " AND ") + " ORDER BY id ASC " + FormatLimitOffset(filter.Limit, filter.Offset))
// 	err := s.conn.SelectContext(ctx, &contests, query, args...)
// 	if errors.Is(err, sql.ErrNoRows) {
// 		return []*kilonova.Contest{}, nil
// 	}
// 	return mapperCtx(ctx, contests, s.internalToContest), err
// }

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

// Contest problems

func (s *DB) UpdateContestProblems(ctx context.Context, contestID int, problems []int) error {
	return s.updateManyToMany(ctx, "contest_problems", "contest_id", "problem_id", contestID, problems, true)
}

// Access rights

func (s *DB) contestAccessRights(ctx context.Context, contestID int) ([]*userAccess, error) {
	return s.getAccess(ctx, "contest_user_access", "contest_id", contestID)
}

func (s *DB) AddContestEditor(ctx context.Context, contestID int, uid int) error {
	return s.addAccess(ctx, "contest_user_access", "contest_id", contestID, uid, "editor")
}

func (s *DB) AddContestTester(ctx context.Context, contestID int, uid int) error {
	return s.addAccess(ctx, "contest_user_access", "contest_id", contestID, uid, "viewer")
}

func (s *DB) StripContestAccess(ctx context.Context, contestID int, uid int) error {
	return s.removeAccess(ctx, "contest_user_access", "contest_id", contestID, uid)
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
	return toUpd, args
}

func (s *DB) internalToContest(ctx context.Context, contest *dbContest) (*kilonova.Contest, error) {

	rights, err := s.contestAccessRights(ctx, contest.ID)
	if err != nil {
		return nil, err
	}

	editors := []*kilonova.UserBrief{}
	viewers := []*kilonova.UserBrief{}
	for _, right := range rights {
		switch right.Access {
		case accessEditor:
			editor, err := s.User(ctx, right.UserID)
			if err != nil {
				zap.S().Warn(err)
				continue
			}
			editors = append(editors, editor.ToBrief())
		case accessViewer:
			viewer, err := s.User(ctx, right.UserID)
			if err != nil {
				zap.S().Warn(right.UserID, contest.ID, err)
				continue
			}
			viewers = append(viewers, viewer.ToBrief())
		default:
			zap.S().Warn("Unknown access rank", zap.String("right", string(right.Access)))
		}
	}

	return &kilonova.Contest{
		ID:         contest.ID,
		CreatedAt:  contest.CreatedAt,
		Name:       contest.Name,
		Editors:    editors,
		Testers:    viewers,
		PublicJoin: contest.PublicJoin,
		StartTime:  contest.StartTime,
		EndTime:    contest.EndTime,
		MaxSubs:    contest.MaxSubCount,

		Visible: contest.Visible,
	}, nil
}
