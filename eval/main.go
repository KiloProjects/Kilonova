package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"

	"github.com/AlexVasiluta/kilonova/datamanager"
	"github.com/AlexVasiluta/kilonova/eval/judge"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

func runMake(args ...string) error {
	args = append(args, "--directory=isolate")
	cmd := exec.Command("make", args...)
	var stdout, stderr bytes.Buffer
	// redirect stdout and stderr
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if len(stdout.String()) > 0 {
		fmt.Println("STDOUT: ", stdout.String())
	}
	if len(stderr.String()) > 0 {
		fmt.Println("STDERR: ", stderr.String())
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

var testLimit = models.Limits{MemoryLimit: 32 * 1024, StackLimit: 16 * 1024, TimeLimit: 1.5}

func main() {
	dataManager := datamanager.NewManager("/home/alexv/Projects/kilonova/data/")

	dataManager.SaveTest(1, 2, []byte(`1 4`), []byte(`5`))

	dataManager.SaveTest(1, 3, []byte(`1 1`), []byte(`2`))

	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	db.AutoMigrate(&models.EvalTest{})
	db.AutoMigrate(&models.Test{})
	db.AutoMigrate(&models.Problem{})
	db.AutoMigrate(&models.Task{})

	pb1 := models.Problem{
		Limits:       testLimit,
		ConsoleInput: true,
		TestName:     "test",
	}
	db.Create(&pb1)

	test1 := models.Test{Score: 20, ProblemID: pb1.ID}
	test2 := models.Test{Score: 20, ProblemID: pb1.ID}
	db.Create(&test1)
	db.Create(&test2)

	gr := judge.NewGrader(context.Background(), db, dataManager)
	err = gr.NewManager(2)
	if err != nil {
		panic(err)
	}
	gr.Start()
	r := chi.NewRouter()
	r.Get("/getTasks", func(w http.ResponseWriter, r *http.Request) {
		var tasks []models.Task
		db.Find(&tasks)
		json.NewEncoder(w).Encode(tasks)
	})
	r.Post("/pushTask", func(w http.ResponseWriter, r *http.Request) {
		lang := r.FormValue("language")
		evtest01 := models.EvalTest{Test: test1}
		evtest02 := models.EvalTest{Test: test2}
		code := r.FormValue("sourcecode")
		if code == "" {
			file, _, err := r.FormFile("file")
			if err != nil {
				fmt.Println(err)
				http.Error(w, "Țeacă", 500)
				return
			}
			var a []byte
			a, err = ioutil.ReadAll(file)
			if err != nil {
				http.Error(w, "Țeacă 2", 500)
				return
			}
			code = string(a)
		}

		db.Create(&evtest01)
		db.Create(&evtest02)
		task := models.Task{
			Language:   lang,
			Problem:    pb1,
			Tests:      []models.EvalTest{evtest01, evtest02},
			SourceCode: code,
		}
		db.Create(&task)
	})
	err = http.ListenAndServe(":8081", r)
	if err != nil {
		fmt.Println(err)
	}
	select {}

}
