package db

import (
	"context"
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
	Hidden      bool      `db:"hidden"`
	StartTime   time.Time `db:"start_time"`
	EndTime     time.Time `db:"end_time"`
	MaxSubCount int       `db:"max_sub_count"`
}

const createContestQuery = `INSERT INTO contest (
		name, public_join, hidden, start_time, end_time
	) VALUES (
		?, ?, ?, ?, ?
	) RETURNING id`

func (s *DB) CreateContest(ctx context.Context, name string, publicJoin, hidden bool) (int, error) {
	if name == "" {
		return -1, kilonova.ErrMissingRequired
	}
	var id int
	// Default is one week from now, 2 hours
	defaultStart := time.Now().AddDate(0, 0, 7).Truncate(time.Hour)
	err := s.conn.GetContext(
		ctx, &id, s.conn.Rebind(createContestQuery),
		name, publicJoin, hidden,
		defaultStart, defaultStart.Add(2*time.Hour),
	)
	return id, err
}

func (s *DB) UpdateContest(ctx context.Context, id int, upd kilonova.ContestUpdate) error {
	toUpd, args := contestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE submissions SET %s WHERE id = ?", strings.Join(toUpd, ", ")))
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
	if v := upd.Hidden; v != nil {
		toUpd, args = append(toUpd, "hidden = ?"), append(args, v)
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
		Hidden:     contest.Hidden,
		StartTime:  contest.StartTime,
		EndTime:    contest.EndTime,
		MaxSubs:    contest.MaxSubCount,
	}, nil
}
