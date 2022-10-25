package eval

import (
	"bytes"
	"context"
	"log"

	"go.uber.org/zap"
)

// RunSubmission runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
func RunSubmission(ctx context.Context, box Sandbox, language Language, constraints Limits, consoleInput bool) (*RunStats, error) {

	var runConf RunConfig
	runConf.EnvToSet = make(map[string]string)

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.Compiled {
		runConf.Directories = append(runConf.Directories, language.Mounts...)
	}

	for key, val := range language.CommonEnv {
		runConf.EnvToSet[key] = val
	}
	for key, val := range language.RunEnv {
		runConf.EnvToSet[key] = val
	}

	runConf.MemoryLimit = constraints.MemoryLimit
	runConf.TimeLimit = constraints.TimeLimit
	runConf.WallTimeLimit = 2*constraints.TimeLimit + 1
	if constraints.TimeLimit == 0 {
		runConf.WallTimeLimit = 30
	}

	if consoleInput {
		runConf.InputPath = "/box/stdin.in"
		runConf.OutputPath = "/box/stdin.out"
	}

	goodCmd, err := MakeGoodCommand(language.RunCommand)
	if err != nil {
		log.Printf("WARNING: function makeGoodCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.RunCommand)
		goodCmd = language.RunCommand
	}

	return box.RunCommand(ctx, goodCmd, &runConf)
}

// CompileFile compiles a file that has the corresponding language
func CompileFile(ctx context.Context, box Sandbox, files map[string][]byte, compiledFiles []string, language Language) (string, error) {
	if files == nil {
		zap.S().Warn("No files specified")
		files = make(map[string][]byte)
	}
	for fileName, fileData := range files {
		if err := box.WriteFile(fileName, bytes.NewReader(fileData), 0644); err != nil {
			return "", err
		}
	}

	var conf RunConfig
	conf.EnvToSet = make(map[string]string)

	conf.InheritEnv = true
	conf.Directories = append(conf.Directories, language.Mounts...)

	for key, val := range language.CommonEnv {
		conf.EnvToSet[key] = val
	}

	for key, val := range language.BuildEnv {
		conf.EnvToSet[key] = val
	}

	goodCmd, err := MakeGoodCompileCommand(language.CompileCommand, compiledFiles)
	if err != nil {
		log.Printf("WARNING: function MakeGoodCompileCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.CompileCommand)
		goodCmd = language.CompileCommand
	}

	var out bytes.Buffer
	conf.Stdout = &out
	conf.Stderr = &out

	_, err = box.RunCommand(ctx, goodCmd, &conf)
	combinedOut := out.String()

	if err != nil {
		return combinedOut, err
	}

	for fileName, fileData := range files {
		if err := box.WriteFile(fileName, bytes.NewReader(fileData), 0644); err != nil {
			return "", err
		}
	}

	return combinedOut, box.RemoveFile(language.SourceName)
}

func MakeGoodCompileCommand(command []string, files []string) ([]string, error) {
	cmd, err := MakeGoodCommand(command)
	if err != nil {
		return nil, err
	}
	for i := range cmd {
		if cmd[i] == MAGIC_REPLACE {
			x := []string{}
			x = append(x, cmd[:i]...)
			x = append(x, files...)
			x = append(x, cmd[i+1:]...)
			return x, nil
		}
	}

	zap.S().Warnf("Didn't replace any fields in %#v", command)
	return cmd, nil
}
