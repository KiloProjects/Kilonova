package main

import (
	"bytes"
	"fmt"
	"log"
	"os/exec"

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
	var testCpp = `#include <stdio.h>
	#include <unistd.h>
	int main()
	{
		printf("Hello from Container, I'm C!");
		return 0;
	}
	`
	var testPy = `print("Hello from Container, I'm python!")`
	bm, err := judge.NewBoxManager(2)
	if err != nil {
		log.Fatalln("Could not create box manager: ", err)
	}

	bm.CompileFile(testCpp, models.Languages["c"])
	bm.RunTask(models.Languages["c"], models.Limits{MemoryLimit: 32 * 1024, StackLimit: 16 * 1024, TimeLimit: 1.1})
	bm.Reset()

	bm.CompileFile(testPy, models.Languages["py"])
	bm.RunTask(models.Languages["py"], models.Limits{})
	bm.Cleanup()

}
