package kilonova

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func ReadKNA(data io.Reader) ([]*FullProblem, error) {
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
			log.Println(err)
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
				log.Println(err)
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
		if err := db.Select(&stks, "SELECT * FROM subtasks WHERE problem_id = ?", problem.ID); err != nil {
			log.Println(err)
			continue
		}
		actualStks := []*SubTask{}
		for _, stk := range stks {
			actualStks = append(actualStks, &SubTask{
				ProblemID: stk.ProblemID,
				VisibleID: stk.VisibleID,
				Score:     stk.Score,
				Tests:     DeserializeIntList(stk.Tests),
			})
		}
		problem.SubTasks = actualStks

		problems = append(problems, &problem)
	}

	return problems, db.Close()
}

func GenKNA(problems []*Problem, testServer TestService, subTaskServer SubTaskService, dm GraderStore) (io.ReadSeekCloser, error) {
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
	helper_code TEXT 		NOT NULL DEFAULT '',
	helper_code_lang TEXT 	NOT NULL DEFAULT 'cpp'
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
		if err := db.Get(&pbid, `INSERT INTO problems (name, description, test_name, time_limit, memory_limit, stack_limit, source_size, console_input, source_credits, author_credits, short_description, default_points, pb_type, helper_code, helper_code_lang) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`, pb.Name, pb.Description, pb.TestName, pb.TimeLimit, pb.MemoryLimit, pb.StackLimit, pb.SourceSize, pb.ConsoleInput, pb.SourceCredits, pb.AuthorCredits, pb.ShortDesc, pb.DefaultPoints, pb.Type, pb.HelperCode, pb.HelperCodeLang); err != nil {
			log.Println(pb.ID, err)
			continue
		}
		tests, err := testServer.Tests(context.Background(), pb.ID)
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

		stks, err := subTaskServer.SubTasks(context.Background(), pb.ID)
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

			if _, err := db.Exec(`INSERT INTO subtasks (problem_id, visible_id, score, tests) VALUES (?, ?, ?, ?)`, pbid, stk.VisibleID, stk.Score, SerializeIntList(newTestIDs)); err != nil {
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
	Problem
	Tests    []*FullTest
	SubTasks []*SubTask
}

type FullTest struct {
	Test
	Input  []byte
	Output []byte
}
