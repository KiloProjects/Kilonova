package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

	"github.com/AlexVasiluta/kilonova/datamanager"
	"github.com/AlexVasiluta/kilonova/eval/judge"
	"github.com/AlexVasiluta/kilonova/models"
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

func main() {

	dataManager := datamanager.NewManager("/home/alexv/Projects/kilonova/data/")

	dataManager.SaveTest(1, 1, []byte(`1 4`), []byte(`5`))

	var testCpp = `
#include <bits/stdc++.h>
using namespace std;
int main()
{
	int n, m;
	cin >> n >> m;
	cout << n + m << "\n\n\n\n\n\n\n\n";
	return 0;
}`
	var testPy = `
n, m = [int(s) for s in input().split()]
print(n + m)`
	bm, err := judge.NewBoxManager(2, dataManager)
	if err != nil {
		log.Fatalln("Could not create box manager: ", err)
	}

	bm.CompileFile(testCpp, models.Languages["cpp"])
	bm.RunTask(models.Languages["cpp"], models.Limits{MemoryLimit: 32 * 1024, StackLimit: 16 * 1024, TimeLimit: 1.1})
	bm.Reset()

	bm.CompileFile(testPy, models.Languages["py"])
	bm.RunTask(models.Languages["py"], models.Limits{})
	bm.Cleanup()

}
