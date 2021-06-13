package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/KiloProjects/kilonova"
)

func (s *DB) SubTestsBySubID(ctx context.Context, subid int) ([]*kilonova.SubTest, error) {
	var subtests []*kilonova.SubTest
	err := s.conn.SelectContext(ctx, &subtests, s.conn.Rebind("SELECT * FROM submission_tests WHERE submission_id = ? ORDER BY id ASC"), subid)
	if errors.Is(err, sql.ErrNoRows) {
		return []*kilonova.SubTest{}, nil
	}
	return subtests, nil
}

func (s *DB) SubTest(ctx context.Context, id int) (*kilonova.SubTest, error) {
	var subtest kilonova.SubTest
	err := s.conn.GetContext(ctx, &subtest, s.conn.Rebind("SELECT * FROM submission_tests WHERE id = ?"), id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return &subtest, err
}

func (s *DB) CreateSubTest(ctx context.Context, subtest *kilonova.SubTest) error {
	if subtest.UserID == 0 || subtest.TestID == 0 || subtest.SubmissionID == 0 {
		return kilonova.ErrMissingRequired
	}
	var id int
	err := s.conn.GetContext(ctx, &id, s.conn.Rebind(`INSERT INTO submission_tests (user_id, test_id, submission_id) VALUES (?, ?, ?) RETURNING id;`), subtest.UserID, subtest.TestID, subtest.SubmissionID)
	if err == nil {
		subtest.ID = id
	}
	return err
}

func (s *DB) UpdateSubTest(ctx context.Context, id int, upd kilonova.SubTestUpdate) error {
	toUpd, args := subtestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, id)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE submission_tests SET %s WHERE id = ?", strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func (s *DB) UpdateSubmissionSubTests(ctx context.Context, subID int, upd kilonova.SubTestUpdate) error {
	toUpd, args := subtestUpdateQuery(&upd)
	if len(toUpd) == 0 {
		return kilonova.ErrNoUpdates
	}
	args = append(args, subID)
	query := s.conn.Rebind(fmt.Sprintf("UPDATE submission_tests SET %s WHERE submission_id = ?", strings.Join(toUpd, ", ")))
	_, err := s.conn.ExecContext(ctx, query, args...)
	return err
}

func subtestUpdateQuery(upd *kilonova.SubTestUpdate) ([]string, []interface{}) {
	toUpd, args := []string{}, []interface{}{}
	if v := upd.Memory; v != nil {
		toUpd, args = append(toUpd, "memory = ?"), append(args, v)
	}
	if v := upd.Score; v != nil {
		toUpd, args = append(toUpd, "score = ?"), append(args, v)
	}
	if v := upd.Time; v != nil {
		toUpd, args = append(toUpd, "time = ?"), append(args, v)
	}
	if v := upd.Verdict; v != nil {
		toUpd, args = append(toUpd, "verdict = ?"), append(args, v)
	}
	if v := upd.Done; v != nil {
		toUpd, args = append(toUpd, "done = ?"), append(args, v)
	}

	return toUpd, args
}
