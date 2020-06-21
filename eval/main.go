package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os/exec"

	"github.com/AlexVasiluta/kilonova/datamanager"
	"github.com/AlexVasiluta/kilonova/eval/judge"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/jinzhu/gorm"
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

var testCpp = `
#include <bits/stdc++.h>
using namespace std;
ifstream f("test.in");
ofstream g("test.out");
int main()
{
	int n, m;
	f >> n >> m;
	g << n + m << "\n\n\n\n\n\n\n\n";
	return 0;
}`
var testPy = `
n, m = [int(s) for s in input().split()]
print(n + m)
`

var testLimit = models.Limits{MemoryLimit: 32 * 1024, StackLimit: 16 * 1024, TimeLimit: 1.1}

func main() {

	dataManager := datamanager.NewManager("/home/alexv/Projects/kilonova/data/")

	dataManager.SaveTest(1, 2, []byte(`1 4`), []byte(`5`))

	dataManager.SaveTest(1, 3, []byte(`1 1`), []byte(`2`))

	bm, err := judge.NewBoxManager(2, dataManager)
	if err != nil {
		log.Fatalln("Could not create box manager: ", err)
	}

	tasks, output := bm.Start(context.Background())

	tasks <- models.Task{
		Language: "cpp",
		Problem: models.Problem{
			Limits:       testLimit,
			ConsoleInput: false,
			TestName:     "test",
		},
		ProblemID: 1,
		Model:     gorm.Model{ID: 123},
		Tests: []models.EvalTest{
			{Model: gorm.Model{ID: 1}, TestID: 2, Test: models.Test{Score: 20}},
			{Model: gorm.Model{ID: 2}, TestID: 3, Test: models.Test{Score: 10}},
		},
		SourceCode: testCpp,
	}
	tasks <- models.Task{
		Language: "py",
		Problem: models.Problem{
			// Limits:       testLimit,
			ConsoleInput: true,
			TestName:     "test",
		},
		ProblemID: 1,
		Model:     gorm.Model{ID: 124},
		Tests: []models.EvalTest{
			{Model: gorm.Model{ID: 1}, TestID: 2, Test: models.Test{Score: 15}},
			{Model: gorm.Model{ID: 2}, TestID: 3, Test: models.Test{Score: 10}},
		},
		SourceCode: testPy,
	}

	go func() {
		for {
			select {
			case out := <-output:
				out.Update(nil)
			}
		}
	}()

	// bm.CompileFile(testCpp, models.Languages["cpp"])
	// bm.RunTask(models.Languages["cpp"], models.Limits{MemoryLimit: 32 * 1024, StackLimit: 16 * 1024, TimeLimit: 1.1}, true)
	// bm.Reset()

	// bm.CompileFile(testPy, models.Languages["py"])
	// bm.RunTask(models.Languages["py"], models.Limits{}, true)
	// bm.Cleanup()

	select {}

}
