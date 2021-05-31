package eval

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/KiloProjects/kilonova/internal/config"
)

const (
	releasePrefix = "https://github.com/KiloProjects/isolate/releases/latest/download/"
	configURL     = releasePrefix + "default.cf"
	configPath    = "/usr/local/etc/isolate"
	isolateURL    = releasePrefix + "isolate"
)

func CopyFromBox(b Sandbox, p string, w io.Writer) error {
	f, err := b.ReadFile(p)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func CopyInBox(b Sandbox, p1 string, p2 string) error {
	file, err := os.Open(p1)
	if err != nil {
		return err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return err
	}

	return b.WriteFile(p2, file, stat.Mode())
}

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
	runConf.StackLimit = constraints.StackLimit
	runConf.TimeLimit = constraints.TimeLimit
	runConf.WallTimeLimit = constraints.TimeLimit + 1
	if constraints.TimeLimit == 0 {
		runConf.WallTimeLimit = 15
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
func CompileFile(ctx context.Context, box Sandbox, SourceCode []byte, language Language) (string, error) {
	if err := box.WriteFile(language.SourceName, bytes.NewReader(SourceCode), 0644); err != nil {
		return "", err
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

	goodCmd, err := MakeGoodCommand(language.CompileCommand)
	if err != nil {
		log.Printf("WARNING: function makeGoodCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.CompileCommand)
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

	return combinedOut, box.RemoveFile(language.SourceName)
}

// makeGoodCommand makes sure it's a full path (with no symlinks) for the command.
// Some languages (like java) are hidden pretty deep in symlinks, and we don't want a hardcoded path that could be different on other platforms.
func MakeGoodCommand(command []string) ([]string, error) {
	tmp := make([]string, len(command))
	copy(tmp, command)

	if strings.HasPrefix(tmp[0], "/box") {
		return tmp, nil
	}

	cmd, err := exec.LookPath(tmp[0])
	if err != nil {
		return nil, err
	}

	cmd, err = filepath.EvalSymlinks(cmd)
	if err != nil {
		return nil, err
	}

	tmp[0] = cmd
	return tmp, nil
}

func CleanCompilation(subid int) error {
	return os.Remove(path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", subid)))
}

func disableLang(key string) {
	lang := Langs[key]
	lang.Disabled = true
	Langs[key] = lang
}

// checkLanguages disables all languages that are *not* detected by the system in the current configuration
// It should be run at the start of the execution (and implemented more nicely tbh)
func checkLanguages() {
	for k, v := range Langs {
		var toSearch []string
		if v.Compiled {
			toSearch = v.CompileCommand
		} else {
			toSearch = v.RunCommand
		}
		if len(toSearch) == 0 {
			disableLang(k)
			log.Printf("Language %q was disabled because of empty line\n", k)
			continue
		}
		cmd, err := exec.LookPath(toSearch[0])
		if err != nil {
			disableLang(k)
			log.Printf("Language %q was disabled because the compiler/interpreter was not found in PATH\n", k)
			continue
		}
		cmd, err = filepath.EvalSymlinks(cmd)
		if err != nil {
			disableLang(k)
			log.Printf("Language %q was disabled because the compiler/interpreter had a bad symlink\n", k)
			continue
		}
		stat, err := os.Stat(cmd)
		if err != nil {
			disableLang(k)
			log.Printf("Language %q was disabled because the compiler/interpreter binary was not found\n", k)
			continue
		}

		if stat.Mode()&0111 == 0 {
			disableLang(k)
			log.Printf("Language %q was disabled because the compiler/interpreter binary is not executable\n", k)
		}

	}
}

// Initialize should be called after reading the flags, but before manager.New
func Initialize() error {

	// Test right now if they exist
	if _, err := os.Stat(config.Eval.IsolatePath); os.IsNotExist(err) {
		// download isolate
		fmt.Println("Downloading isolate binary")
		if err := downloadFile(isolateURL, config.Eval.IsolatePath, 0744); err != nil {
			return err
		}
		fmt.Println("Isolate binary downloaded")
	}
	if _, err := os.Stat(config.Eval.IsolatePath); os.IsNotExist(err) {
		// download the config file
		fmt.Println("Downloading isolate config")
		if err := downloadFile(configURL, config.Eval.IsolatePath, 0644); err != nil {
			return err
		}
		fmt.Println("Isolate config downloaded")
	}

	if err := os.MkdirAll(config.Eval.CompilePath, 0777); err != nil {
		return err
	}

	checkLanguages()

	return nil
}

func downloadFile(url, path string, perm os.FileMode) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return file.Chmod(perm)
}
