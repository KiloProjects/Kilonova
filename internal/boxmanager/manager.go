package boxmanager

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"golang.org/x/sync/semaphore"
)

var _ kilonova.Runner = &BoxManager{}

// limits stores the constraints that need to be respected by a submission
type limits struct {
	// seconds
	TimeLimit float64
	// kilobytes
	StackLimit  int
	MemoryLimit int
}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	dm kilonova.DataStore

	numConcurrent int
	sem           *semaphore.Weighted

	availableIDs chan int

	// If debug mode is enabled, the manager should print more stuff to the command line
	debug bool
}

// ToggleDebug is a convenience function to setting up debug mode in the box manager and all future boxes
// It should print additional output
func (b *BoxManager) ToggleDebug() {
	b.debug = !b.debug
}

// CompileFile compiles a file that has the corresponding language
func (b *BoxManager) CompileFile(ctx context.Context, box *Box, SourceCode []byte, language config.Language) (string, error) {
	if err := box.WriteFile(language.SourceName, bytes.NewReader(SourceCode)); err != nil {
		return "", err
	}

	var conf kilonova.RunConfig
	conf.EnvToSet = make(map[string]string)

	conf.InheritEnv = true
	conf.Directories = append(conf.Directories, language.Mounts...)

	for key, val := range language.CommonEnv {
		conf.EnvToSet[key] = val
	}

	for key, val := range language.BuildEnv {
		conf.EnvToSet[key] = val
	}

	goodCmd, err := makeGoodCommand(language.CompileCommand)
	if err != nil {
		log.Printf("WARNING: function makeGoodCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.CompileCommand)
		goodCmd = language.CompileCommand
	}

	combinedOut, err := box.ExecCombinedOutput(ctx, goodCmd, &conf)

	if err != nil {
		return string(combinedOut), err
	}

	return string(combinedOut), box.RemoveFile(language.SourceName)
}

// RunSubmission runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
func (b *BoxManager) RunSubmission(ctx context.Context, box *Box, language config.Language, constraints limits, consoleInput bool) (*kilonova.RunStats, error) {
	var runConf kilonova.RunConfig
	runConf.EnvToSet = make(map[string]string)

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.IsCompiled {
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

	goodCmd, err := makeGoodCommand(language.RunCommand)
	if err != nil {
		log.Printf("WARNING: function makeGoodCommand returned an error: %q. This is not good, so we'll use the command from the config file. The supplied command was %#v", err, language.RunCommand)
		goodCmd = language.RunCommand
	}

	return box.RunCommand(ctx, goodCmd, &runConf)
}

func (b *BoxManager) Execute(ctx context.Context, sub *kilonova.ExecRequest) (*kilonova.ExecResponse, error) {
	response := &kilonova.ExecResponse{SubtestID: sub.SubtestID}

	box, err := b.GetSandbox(ctx)
	if err != nil {
		return response, err
	}

	// After doing stuff, we need to clean up after ourselves ;)
	defer b.ReleaseSandbox(box)

	if b.debug {
		log.Printf("Executing test %d using box %d\n", sub.SubtestID, box.boxID)
	}

	in, err := b.dm.TestInput(int(sub.TestID))
	if err != nil {
		return response, err
	}
	defer in.Close()

	if err := box.WriteFile("/box/"+sub.Filename+".in", in); err != nil {
		fmt.Println("Can't write input file:", err)
		response.Comments = "Sandbox error: Couldn't write input file"
		return response, err
	}
	consoleInput := sub.Filename == "stdin"

	lang := config.Languages[sub.Lang]
	if err := box.CopyInBox(path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", sub.SubID)), lang.CompiledName); err != nil {
		response.Comments = "Couldn't link executable in box"
		return response, err
	}

	lim := limits{
		MemoryLimit: int(sub.MemoryLimit),
		StackLimit:  int(sub.StackLimit),
		TimeLimit:   sub.TimeLimit,
	}
	meta, err := b.RunSubmission(ctx, box, config.Languages[sub.Lang], lim, consoleInput)
	if err != nil {
		response.Comments = fmt.Sprintf("Error running submission: %v", err)
		return response, nil
	}
	response.Time = meta.Time
	response.Memory = meta.Memory

	switch meta.Status {
	case "TO":
		response.Comments = "TLE: " + meta.Message
	case "RE":
		response.Comments = "Runtime Error: " + meta.Message
	case "SG":
		response.Comments = meta.Message
	case "XX":
		response.Comments = "Sandbox Error: " + meta.Message
	}

	boxOut := fmt.Sprintf("/box/%s.out", sub.Filename)
	if !box.FileExists(boxOut) {
		response.Comments = "No output file found"
		return response, nil
	}

	w, err := b.dm.SubtestWriter(sub.SubtestID)
	if err != nil {
		response.Comments = "Could not open problem output"
		return response, nil
	}

	if err := box.CopyFromBox(boxOut, w); err != nil {
		response.Comments = "Could not write output file"
		return response, nil
	}

	if err := w.Close(); err != nil {
		response.Comments = "Could not close output file"
		return response, nil
	}

	return response, nil
}

func (b *BoxManager) Compile(ctx context.Context, c *kilonova.CompileRequest) (*kilonova.CompileResponse, error) {
	box, err := b.GetSandbox(ctx)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	defer b.ReleaseSandbox(box)

	if b.debug {
		log.Printf("Compiling file using box %d\n", box.boxID)
	}

	lang := config.Languages[c.Lang]

	outName := path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", c.ID))
	resp := &kilonova.CompileResponse{}
	resp.Success = true

	if lang.IsCompiled {
		out, err := b.CompileFile(ctx, box, c.Code, lang)
		resp.Output = out

		if err != nil {
			resp.Success = false
		} else {
			f, err := os.OpenFile(outName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0777)
			if err != nil {
				resp.Other = err.Error()
				resp.Success = false
				return resp, nil
			}
			if err := box.CopyFromBox(lang.CompiledName, f); err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
			if err := f.Close(); err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
		}

		return resp, nil
	}

	if err := os.WriteFile(outName, c.Code, 0644); err != nil {
		resp.Other = err.Error()
		resp.Success = false
	}

	return resp, nil
}

func (b *BoxManager) NewSandbox() (*Box, error) {
	box, err := newBox(<-b.availableIDs)
	if err != nil {
		return nil, err
	}
	box.Debug = b.debug
	return box, nil
}

func (b *BoxManager) GetSandbox(ctx context.Context) (*Box, error) {
	if err := b.sem.Acquire(ctx, 1); err != nil {
		return nil, err
	}
	return b.NewSandbox()
}

func (b *BoxManager) ReleaseSandbox(sb *Box) {
	b.availableIDs <- sb.boxID
	b.sem.Release(1)
	if err := sb.Close(); err != nil {
		log.Printf("Could not release sandbox %d: %v\n", sb.boxID, err)
	}
}

func (b *BoxManager) Clean(ctx context.Context, subid int) error {
	p := path.Join(config.Eval.CompilePath, fmt.Sprintf("%d.bin", subid))
	return os.Remove(p)
}

func (b *BoxManager) Close(ctx context.Context) error {
	b.sem.Acquire(ctx, int64(b.numConcurrent))
	close(b.availableIDs)
	return nil
}

// New creates a new box manager
func New(count int, dm kilonova.DataStore) (*BoxManager, error) {

	sem := semaphore.NewWeighted(int64(count))

	availableIDs := make(chan int, 3*count)
	for i := 1; i <= 2*count; i++ {
		availableIDs <- i
	}

	bm := &BoxManager{
		dm:            dm,
		sem:           sem,
		availableIDs:  availableIDs,
		numConcurrent: count,
	}
	return bm, nil
}

// makeGoodCommand makes sure it's a full path (with no symlinks) for the command.
// Some languages (like java) are hidden pretty deep in symlinks, and we don't want a hardcoded path that could be different on other platforms.
func makeGoodCommand(command []string) ([]string, error) {
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

func disableLang(key string) {
	lang := config.Languages[key]
	lang.Disabled = true
	config.Languages[key] = lang
}

// CheckLanguages disables all languages that are *not* detected by the system in the current configuration
// It should be run at the start of the execution (and implemented more nicely tbh)
func CheckLanguages() {
	for k, v := range config.Languages {
		var toSearch []string
		if v.IsCompiled {
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
