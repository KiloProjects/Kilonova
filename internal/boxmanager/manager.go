package boxmanager

import (
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/config"
	pb "github.com/KiloProjects/Kilonova/internal/grpc"
)

var compilePath string

// limits stores the constraints that need to be respected by a submission
type limits struct {
	// seconds
	TimeLimit float64
	// kilobytes
	StackLimit  int32
	MemoryLimit int32
}

// BoxManager manages a box with eval-based submissions
type BoxManager struct {
	ID  int
	Box *box.Box
	dm  datamanager.Manager

	compileLock   sync.Mutex
	executionLock sync.Mutex

	// If debug mode is enabled, the manager should print more stuff to the command line
	debug bool
}

// ToggleDebug is a convenience function to setting up debug mode in the box and the box manager
// It should print additional output
func (b *BoxManager) ToggleDebug() {
	b.debug = !b.debug
	b.Box.Debug = b.debug
}

// Cleanup cleans up the boxes
func (b *BoxManager) Cleanup() error {
	return b.Box.Cleanup()
}

// CompileFile compiles a file that has the corresponding language
func (b *BoxManager) CompileFile(SourceCode []byte, language config.Language) (string, error) {
	if err := b.Box.WriteFile(language.SourceName, SourceCode); err != nil {
		return "", err
	}

	if b.Box.Config.EnvToSet == nil {
		b.Box.Config.EnvToSet = make(map[string]string)
	}

	oldConfig := b.Box.Config
	b.Box.Config.InheritEnv = true
	for _, dir := range language.Mounts {
		b.Box.Config.Directories = append(b.Box.Config.Directories, dir)
	}

	for key, val := range language.CommonEnv {
		b.Box.Config.EnvToSet[key] = val
	}

	for key, val := range language.BuildEnv {
		b.Box.Config.EnvToSet[key] = val
	}

	combinedOut, err := b.Box.ExecCombinedOutput(language.CompileCommand...)
	b.Box.Config = oldConfig

	if err != nil {
		return string(combinedOut), err
	}

	return string(combinedOut), b.Box.RemoveFile(language.SourceName)
}

// RunSubmission runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
func (b *BoxManager) RunSubmission(language config.Language, constraints limits, metaFile string, consoleInput bool) (*box.MetaFile, error) {
	if b.Box.Config.EnvToSet == nil {
		b.Box.Config.EnvToSet = make(map[string]string)
	}

	oldConf := b.Box.Config

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.IsCompiled {
		for _, dir := range language.Mounts {
			b.Box.Config.Directories = append(b.Box.Config.Directories, dir)
		}
	}

	for key, val := range language.CommonEnv {
		b.Box.Config.EnvToSet[key] = val
	}
	for key, val := range language.RunEnv {
		b.Box.Config.EnvToSet[key] = val
	}

	//b.Box.Config.MemoryLimit = constraints.MemoryLimit
	// CgroupMem is disabled for now, it causes a sandbox error "Cannot set /sys/fs/cgroup/memory/box-2/memory.limit_in_bytes"
	// and i don't want to deal with it right now
	b.Box.Config.CgroupMem = constraints.MemoryLimit
	b.Box.Config.StackSize = constraints.StackLimit
	b.Box.Config.TimeLimit = constraints.TimeLimit
	// give a little bit more wall time
	b.Box.Config.WallTimeLimit = constraints.TimeLimit + 1
	if constraints.TimeLimit == 0 {
		// set a hard limit at 15 seconds if no time is specified
		b.Box.Config.WallTimeLimit = 15
	}

	if metaFile != "" {
		metaFile = path.Join("/tmp/", metaFile)
		b.Box.Config.MetaFile = metaFile
	}

	if consoleInput {
		b.Box.Config.InputFile = "/box/stdin.in"
		b.Box.Config.OutputFile = "/box/stdin.out"
	}

	defer func() {
		b.Box.Config = oldConf
	}()

	_, _, err := b.Box.ExecCommand(language.RunCommand...)
	if metaFile != "" {
		f, err := os.Open(metaFile)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return box.ParseMetaFile(f), nil
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ExecuteTest executes a new test
func (b *BoxManager) ExecuteTest(sub *pb.Test) (*pb.TestResponse, error) {
	b.executionLock.Lock()
	defer b.executionLock.Unlock()

	defer func() {
		// After doing stuff, we need to clean up after ourselves ;)
		if err := b.Reset(); err != nil {
			fmt.Printf("CAN'T RESET BOX %d: %d", b.ID, err)
		}
	}()

	response := &pb.TestResponse{TID: sub.TID}

	in, err := b.dm.TestInput(sub.TestID)
	if err != nil {
		return response, err
	}
	defer in.Close()

	if err := b.Box.WriteReader("/box/"+sub.Filename+".in", in); err != nil {
		fmt.Println("Can't write input file:", err)
		response.Comments = "Sandbox error: Couldn't write input file"
		return response, err
	}
	consoleInput := sub.Filename == "stdin"

	lang := config.C.Languages[sub.Lang]
	if err := b.Box.CopyInBox(path.Join(compilePath, fmt.Sprintf("%d.bin", sub.ID)), lang.CompiledName); err != nil {
		response.Comments = "Couldn't link executable in box"
		return response, err
	}

	lim := limits{
		MemoryLimit: sub.MemoryLimit,
		StackLimit:  sub.StackLimit,
		TimeLimit:   sub.TimeLimit,
	}
	meta, err := b.RunSubmission(config.C.Languages[sub.Lang], lim, strconv.Itoa(int(sub.ID))+".txt", consoleInput)
	response.Time = meta.Time
	response.Memory = int32(meta.CgMem)

	if err != nil {
		response.Comments = fmt.Sprintf("Error running submission: %v", err)
		return response, err
	}

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
	if _, err := b.Box.Stat(boxOut); err != nil {
		response.Comments = "No output file found"
		return response, nil
	}

	w, err := b.dm.SubtestWriter(sub.TID)
	if err != nil {
		response.Comments = "Could not open problem output"
		return response, nil
	}

	if err := b.Box.CopyFromBoxInWriter(boxOut, w); err != nil {
		response.Comments = "Could not write output file"
		return response, nil
	}

	if err := w.Close(); err != nil {
		response.Comments = "Could not close output file"
		return response, nil
	}

	return response, nil
}

func (b *BoxManager) CompileSubmission(c *pb.CompileRequest) (*pb.CompileResponse, error) {
	b.compileLock.Lock()
	defer b.compileLock.Unlock()

	defer b.Reset()
	lang := config.C.Languages[c.Lang]

	outName := path.Join(compilePath, fmt.Sprintf("%d.bin", c.ID))
	resp := &pb.CompileResponse{}
	resp.Success = true

	if lang.IsCompiled {
		out, err := b.CompileFile([]byte(c.Code), lang)
		resp.Output = out

		if err != nil {
			resp.Success = false
		} else {
			if err := b.Box.CopyFromBox(lang.CompiledName, outName); err != nil {
				resp.Other = err.Error()
				resp.Success = false
			}
		}

		return resp, nil
	}

	if err := os.WriteFile(outName, []byte(c.Code), 0644); err != nil {
		resp.Other = err.Error()
		resp.Success = false
	}

	return resp, nil
}

// Reset reintializes a box
// Should be run after finishing running a batch of tests
func (b *BoxManager) Reset() (err error) {
	err = b.Box.Cleanup()
	if err != nil {
		return
	}
	b.Box, err = box.New(box.Config{ID: b.ID, Cgroups: true})
	b.Box.Debug = b.debug
	return
}

func SetCompilePath(path string) {
	compilePath = path
}

// New creates a new box manager
func New(id int, dm datamanager.Manager) (*BoxManager, error) {
	b, err := box.New(box.Config{ID: id, Cgroups: true})
	if err != nil {
		return nil, err
	}
	b.Config.EnvToSet = make(map[string]string)

	bm := &BoxManager{
		ID:  id,
		Box: b,
		dm:  dm,
	}
	return bm, nil
}
