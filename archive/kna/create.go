package kna

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/KiloProjects/kilonova"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func Generate(problems []*kilonova.Problem, outDB kilonova.DB, dm kilonova.GraderStore) (io.ReadSeekCloser, error) {
	// Creating the file
	file, err := os.CreateTemp("", "kna_w-*.db")
	if err != nil {
		return nil, err
	}
	path := file.Name()
	file.Close()

	// Opening the file
	db, err := sqlx.Connect("sqlite3", "file:"+path+"?_fk=on&mode=rwc")
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS problems (
	id 			INTEGER 	PRIMARY KEY,
	name 		TEXT 		NOT NULL UNIQUE,
	description TEXT 		NOT NULL DEFAULT '',
	test_name 	TEXT 		NOT NULL,
	time_limit 	FLOAT 		NOT NULL DEFAULT 0.1,
	memory_limit INTEGER 	NOT NULL DEFAULT 65536,
	stack_limit INTEGER 	NOT NULL DEFAULT 16384,

	source_size INTEGER 	NOT NULL DEFAULT 10000,
	console_input INTEGER 	NOT NULL DEFAULT FALSE,

	source_credits TEXT 	NOT NULL DEFAULT '',
	author_credits TEXT 	NOT NULL DEFAULT '',
	short_description TEXT 	NOT NULL DEFAULT '',
	default_points INTEGER 	NOT NULL DEFAULT 0,

	pb_type 	TEXT CHECK(pb_type IN ('classic', 'interactive', 'custom_checker')) NOT NULL DEFAULT 'classic',
);`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS tests (
	id 			INTEGER 	PRIMARY KEY,
	score 		INTEGER 	NOT NULL,
	problem_id 	INTEGER 	NOT NULL REFERENCES problems(id),
	visible_id 	INTEGER 	NOT NULL
);`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS test_inputs (
	test_id 	INTEGER 	NOT NULL,
	data 		BLOB 		NOT NULL
);`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS test_outputs (
	test_id 	INTEGER 	NOT NULL,
	data 		BLOB 		NOT NULL
);`); err != nil {
		return nil, err
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS subtasks (
	problem_id 	INTEGER 	NOT NULL REFERENCES problems(id),
	visible_id 	INTEGER 	NOT NULL,
	score 		INTEGER 	NOT NULL,
	tests 		TEXT 		NOT NULL
);`); err != nil {
		return nil, err
	}

	for _, pb := range problems {
		var pbid int
		if err := db.Get(&pbid, `INSERT INTO problems (name, description, test_name, time_limit, memory_limit, stack_limit, source_size, console_input, source_credits, author_credits, short_description, default_points, pb_type) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`, pb.Name, pb.Description, pb.TestName, pb.TimeLimit, pb.MemoryLimit, pb.StackLimit, pb.SourceSize, pb.ConsoleInput, pb.SourceCredits, pb.AuthorCredits, pb.ShortDesc, pb.DefaultPoints, pb.Type); err != nil {
			log.Println(pb.ID, err)
			continue
		}
		tests, err := outDB.Tests(context.Background(), pb.ID)
		if err != nil {
			log.Println(err)
			continue
		}

		testAssocs := make(map[int]int)

		for _, test := range tests {
			var testid int
			if err := db.Get(&testid, `INSERT INTO tests (score, problem_id, visible_id) VALUES (?, ?, ?) RETURNING id`, test.Score, pbid, test.VisibleID); err != nil {
				log.Println(pb.ID, test.ID, err)
				continue
			}
			testAssocs[test.ID] = testid
			dataReader, err := dm.TestInput(test.ID)
			if err != nil {
				continue
			}
			data, err := io.ReadAll(dataReader)
			if err != nil {
				continue
			}
			if _, err := db.Exec(`INSERT INTO test_inputs (test_id, data) VALUES (?, ?)`, testid, data); err != nil {
				continue
			}

			dataReader, err = dm.TestOutput(test.ID)
			if err != nil {
				continue
			}
			data, err = io.ReadAll(dataReader)
			if err != nil {
				continue
			}
			if _, err := db.Exec(`INSERT INTO test_outputs (test_id, data) VALUES (?, ?)`, testid, data); err != nil {
				continue
			}
		}

		stks, err := outDB.SubTasks(context.Background(), pb.ID)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, stk := range stks {
			newTestIDs := []int{}
			for _, id := range stk.Tests {
				newID, ok := testAssocs[id]
				if ok {
					newTestIDs = append(newTestIDs, newID)
				}
			}

			if _, err := db.Exec(`INSERT INTO subtasks (problem_id, visible_id, score, tests) VALUES (?, ?, ?, ?)`, pbid, stk.VisibleID, stk.Score, kilonova.SerializeIntList(newTestIDs)); err != nil {
				log.Println(pb.ID, stk.ID, err)
				continue
			}
		}
	}

	// Writing to io.Writer
	if err := db.Close(); err != nil {
		return nil, err
	}

	return os.Open(path)
}

type knasubtask struct {
	ProblemID int    `db:"problem_id"`
	VisibleID int    `db:"visible_id"`
	Score     int    `db:"score"`
	Tests     string `db:"tests"`
}

type FullProblem struct {
	kilonova.Problem
	Tests    []*FullTest
	SubTasks []*kilonova.SubTask
}

type FullTest struct {
	kilonova.Test
	Input  []byte
	Output []byte
}
