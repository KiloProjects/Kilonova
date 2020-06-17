package main

import (
	"bytes"
	"fmt"
	"os/exec"

	"github.com/AlexVasiluta/kilonova/isolation/box"
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

var testCode string = `#include <stdio.h>

int main()
{
	printf("Hello from container!\n");
	return 0;
}
`

func main() {
	_, err := exec.LookPath("isolate")
	if err != nil && err.(*exec.Error).Err == exec.ErrNotFound {
		fmt.Println("Compiling isolate")
		runMake()
		fmt.Println("Installing isolate")
		runMake("install")
		fmt.Println("Cleaning up compilation")
		runMake("clean")
		fmt.Println("Finished with isolate")
	}
	b := box.NewBox(box.Config{
		ID:          0,
		Cgroups:     true,
		InheritEnv:  true,
		Directories: []box.Directory{{In: "/etc", Out: "/etc"}},
	})
	b.WriteFile("box/file.c", testCode)
	out, _ := b.ExecWithStdin(testCode, "/usr/bin/g++", "-std=c++11", "-O2", "-pipe", "-s", "/box/file.c")
	fmt.Println(out)
	out, _ = b.ExecWithStdin(testCode, "a.out")
	fmt.Println(out)
	// out, _ := b.ExecWithStdin(testCode, "/bin/bash", "-c", "echo $PATH")
	// reader := bufio.NewReader(os.Stdin)
	// for {
	// 	text, _ := reader.ReadString('\n')
	// 	text = strings.TrimSpace(text)
	// 	fmt.Printf("`%s`\n", text)
	// 	if text == "exit" {
	// 		break
	// 	}
	// 	out, _ := b.ExecWithStdin(testCode, "/bin/bash", "-c", text)
	//  fmt.Println(out)
	// }
	b.Cleanup()
}
