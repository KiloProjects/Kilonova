package db

import (
	"context"
	"errors"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (s *DB) ProblemList(ctx context.Context, id int) (*kilonova.ProblemList, error) {
	rows, _ := s.conn.Query(ctx, `
	SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
		FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
		WHERE id = $1 LIMIT 1`, id)
	pblist, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[pblist])
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToPbList(ctx, &pblist)
}

func (s *DB) ProblemListByName(ctx context.Context, name string) (*kilonova.ProblemList, error) {
	rows, _ := s.conn.Query(ctx, `
	SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
		FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
		WHERE title = $1 ORDER BY id LIMIT 1`, name)
	pblist, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[pblist])
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s.internalToPbList(ctx, &pblist)
}

func (s *DB) ProblemLists(ctx context.Context, root bool) ([]*kilonova.ProblemList, error) {
	q := `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
	FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
	ORDER BY lists.id ASC`
	if root {
		q = `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
		FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
		WHERE NOT EXISTS (SELECT 1 FROM problem_list_pblists WHERE child_id = lists.id) ORDER BY lists.id ASC`
	}
	rows, _ := s.conn.Query(ctx, q)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[pblist])
	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) ProblemListsByProblemID(ctx context.Context, problemID int, showHidable bool) ([]*kilonova.ProblemList, error) {
	hidableQ := ""
	if !showHidable {
		hidableQ = " AND lists.sidebar_hidable = false "
	}

	q := `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
	FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
	WHERE EXISTS (SELECT 1 FROM problem_list_problems WHERE pblist_id = lists.id AND problem_id = $1)` + hidableQ + ` ORDER BY lists.id ASC`
	rows, _ := s.conn.Query(ctx, q, problemID)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[pblist])

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) ParentProblemListsByPblistID(ctx context.Context, pblistID int) ([]*kilonova.ProblemList, error) {
	q := `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
	FROM problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id 
	WHERE EXISTS (SELECT 1 FROM problem_list_pblists WHERE parent_id = lists.id AND child_id = $1) ORDER BY lists.id ASC`
	rows, _ := s.conn.Query(ctx, q, pblistID)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[pblist])

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) ChildrenProblemListsByPblistID(ctx context.Context, pblistID int) ([]*kilonova.ProblemList, error) {
	var lists []*pblist

	q := `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
	FROM (problem_lists lists LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id)
		INNER JOIN problem_list_pblists plpp ON (plpp.parent_id = $1 AND plpp.child_id = lists.id)
	ORDER BY plpp.position ASC, lists.id ASC`
	rows, _ := s.conn.Query(ctx, q, pblistID)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[pblist])

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ProblemList{}, nil
	} else if err != nil {
		return []*kilonova.ProblemList{}, err
	}

	return mapperCtx(ctx, lists, s.internalToPbList), nil
}

func (s *DB) shallowProblemLists(ctx context.Context, parentID int) ([]*kilonova.ShallowProblemList, error) {
	query := `SELECT lists.*, COALESCE(cnt.count, 0) AS num_problems, array(SELECT problem_id FROM problem_list_problems WHERE pblist_id = lists.id ORDER BY position ASC, problem_id ASC)::int[] AS list_problems
	FROM (problem_lists lists INNER JOIN problem_list_pblists plpb ON plpb.parent_id = $1 AND lists.id = plpb.child_id)
		LEFT JOIN problem_list_pb_count cnt ON cnt.list_id = lists.id
	ORDER BY plpb.position ASC, lists.id ASC`
	rows, _ := s.conn.Query(ctx, query, parentID)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[pblist])

	if errors.Is(err, pgx.ErrNoRows) || errors.Is(err, context.Canceled) {
		return []*kilonova.ShallowProblemList{}, nil
	}

	outLists := make([]*kilonova.ShallowProblemList, 0, len(lists))
	for _, el := range lists {
		pblist, err := s.internalToShallowProblemList(el)
		if err != nil {
			zap.S().Warn(err)
			continue
		}
		outLists = append(outLists, pblist)
	}
	return outLists, err
}

func (s *DB) NumSolvedPblistProblems(ctx context.Context, listID, userID int) (int, error) {
	var cnt int
	// MAX(count) is a hacky fix for when there's no rows
	// TODO: proper fix
	err := s.conn.QueryRow(ctx, `SELECT COALESCE(MAX(count), 0) FROM pblist_user_solved WHERE list_id = $1 AND user_id = $2`, listID, userID).Scan(&cnt)
	if err != nil {
		zap.S().Warn(err)
		return -1, err
	}
	return cnt, nil
}

func (s *DB) NumBulkedSolvedPblistProblems(ctx context.Context, userID int, listIDs []int) (map[int]int, error) {
	rows, _ := s.conn.Query(ctx, "SELECT list_id, count FROM pblist_user_solved WHERE user_id = $1 AND list_id = ANY($2)", userID, listIDs)
	lists, err := pgx.CollectRows(rows, pgx.RowToAddrOfStructByName[struct {
		ListID int `db:"list_id"`
		Count  int `db:"count"`
	}])
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	var rez = make(map[int]int)
	for _, val := range lists {
		rez[val.ListID] = val.Count
	}

	return rez, nil
}

const createProblemListQuery = "INSERT INTO problem_lists (author_id, title, description, sidebar_hidable) VALUES ($1, $2, $3, $4) RETURNING id;"

func (s *DB) CreateProblemList(ctx context.Context, list *kilonova.ProblemList) error {
	if list.AuthorID == 0 {
		return kilonova.ErrMissingRequired
	}
	// Do insertion
	var id int
	err := s.conn.QueryRow(ctx, createProblemListQuery, list.AuthorID, list.Title, list.Description, list.SidebarHidable).Scan(&id)
	if err != nil {
		return err
	}
	list.ID = id

	// Add problems
	return s.UpdateProblemListProblems(ctx, list.ID, list.List)
}

func (s *DB) UpdateProblemList(ctx context.Context, id int, upd kilonova.ProblemListUpdate) error {
	ub := newUpdateBuilder()
	if v := upd.AuthorID; v != nil {
		ub.AddUpdate("author_id = %s", v)
	}
	if v := upd.Title; v != nil {
		ub.AddUpdate("title = %s", v)
	}
	if v := upd.Description; v != nil {
		ub.AddUpdate("description = %s", v)
	}
	if v := upd.SidebarHidable; v != nil {
		ub.AddUpdate("sidebar_hidable = %s", v)
	}
	if ub.CheckUpdates() != nil {
		return ub.CheckUpdates()
	}
	fb := ub.MakeFilter()
	fb.AddConstraint("id = %s", id)
	_, err := s.conn.Exec(ctx, `UPDATE problem_lists SET `+fb.WithUpdate(), fb.Args()...)
	return err
}

func (s *DB) DeleteProblemList(ctx context.Context, id int) error {
	_, err := s.conn.Exec(ctx, "DELETE FROM problem_lists WHERE id = $1", id)
	return err
}

func (s *DB) UpdateProblemListProblems(ctx context.Context, id int, problemIDs []int) error {
	return s.updateManyToMany(ctx, "problem_list_problems", "pblist_id", "problem_id", id, problemIDs, true)
}

func (s *DB) UpdateProblemListSublists(ctx context.Context, id int, listIDs []int) error {
	// Quick sanity check first
	for _, listID := range listIDs {
		if id == listID {
			return kilonova.Statusf(400, "List %d cannot nest itself!", id)
		}
	}

	return s.updateManyToMany(ctx, "problem_list_pblists", "parent_id", "child_id", id, listIDs, true)
}

type pblist struct {
	ID          int
	CreatedAt   time.Time `db:"created_at"`
	AuthorID    int       `db:"author_id"`
	Title       string
	Description string

	SidebarHidable bool `db:"sidebar_hidable"`

	NumProblems int `db:"num_problems"`

	ListProblems []int `db:"list_problems"`
}

func (s *DB) internalToPbList(ctx context.Context, list *pblist) (*kilonova.ProblemList, error) {
	pblist := &kilonova.ProblemList{
		ID:          list.ID,
		CreatedAt:   list.CreatedAt,
		Title:       list.Title,
		Description: list.Description,
		AuthorID:    list.AuthorID,
		NumProblems: list.NumProblems,

		SidebarHidable: list.SidebarHidable,
	}
	if list.ListProblems != nil {
		pblist.List = list.ListProblems
	} else {
		pblist.List = []int{}
		zap.S().WithOptions(zap.AddCallerSkip(1)).Info("Forgot to query for list problems")
	}

	var err error
	pblist.SubLists, err = s.shallowProblemLists(ctx, list.ID)
	if err != nil {
		return nil, err
	}

	return pblist, nil
}

func (s *DB) internalToShallowProblemList(list *pblist) (*kilonova.ShallowProblemList, error) {
	return &kilonova.ShallowProblemList{
		ID:       list.ID,
		Title:    list.Title,
		AuthorID: list.AuthorID,

		SidebarHidable: list.SidebarHidable,

		NumProblems: list.NumProblems,
	}, nil
}
