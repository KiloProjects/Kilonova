package kna

import (
	"database/sql"
	"errors"
	"io"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

func Parse(data io.Reader) ([]*FullProblem, error) {
	// Writing the file to disk
	file, err := os.CreateTemp("", "kna_r-*.db")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	_, err = io.Copy(file, data)
	err1 := file.Close()
	if err == nil && err1 != nil {
		err = err1
	}
	if err != nil {
		return nil, err
	}

	db, err := sqlx.Connect("sqlite3", "file:"+path+"?_fk=on&mode=ro&immutable=on")
	if err != nil {
		return nil, err
	}

	pbrows, err := db.Queryx("SELECT * FROM problems;")
	if err != nil {
		return nil, err
	}

	problems := []*FullProblem{}
	for pbrows.Next() {
		var problem FullProblem
		if err := pbrows.StructScan(&problem.Problem); err != nil {
			zap.S().Warn("Couldn't StructScan problem", err)
			continue
		}

		testrows, err := db.Queryx("SELECT * FROM tests WHERE problem_id = ?", problem.ID)
		if err != nil {
			continue
		}

		problem.Tests = []*FullTest{}
		for testrows.Next() {
			var test FullTest
			if err := testrows.StructScan(&test.Test); err != nil {
				zap.S().Warn("Couldn't StructScan test", err)
				continue
			}

			input := []byte{}
			err := db.Get(&input, "SELECT data FROM test_inputs WHERE test_id = ?", test.ID)
			if err != nil {
				continue
			}

			output := []byte{}
			err = db.Get(&output, "SELECT data FROM test_outputs WHERE test_id = ?", test.ID)
			if err != nil {
				continue
			}

			test.Input = input
			test.Output = output
			problem.Tests = append(problem.Tests, &test)
		}

		stks := []*knasubtask{}
		if err := db.Select(&stks, "SELECT * FROM subtasks WHERE problem_id = ?", problem.ID); err != nil && !errors.Is(err, sql.ErrNoRows) {
			zap.S().Warn("Couldn't get subtasks", err)
			continue
		}
		actualStks := []*kilonova.SubTask{}
		for _, stk := range stks {
			actualStks = append(actualStks, &kilonova.SubTask{
				ProblemID: stk.ProblemID,
				VisibleID: stk.VisibleID,
				Score:     stk.Score,
				Tests:     kilonova.DeserializeIntList(stk.Tests),
			})
		}
		problem.SubTasks = actualStks

		problems = append(problems, &problem)
	}

	return problems, db.Close()
}
