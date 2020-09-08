package manager

import (
	"fmt"
	"io/ioutil"
	"path"
	"strconv"

	"github.com/KiloProjects/Kilonova/internal/box"
	"github.com/KiloProjects/Kilonova/internal/proto"
)

var compilePath string

// limits stores the constraints that need to be respected by a task
// this has been moved from common because I want to remove another dependency
type limits struct {
	// seconds
	TimeLimit float64
	// kilobytes
	StackLimit  int
	MemoryLimit int
}

// BoxManager manages a box with eval-based tasks
type BoxManager struct {
	ID  int
	Box *box.Box

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
func (b *BoxManager) CompileFile(SourceCode []byte, language proto.Language) (string, error) {
	if err := b.Box.WriteFile(language.SourceName, SourceCode); err != nil {
		return "", err
	}

	/* ***/

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

// RunTask runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
func (b *BoxManager) RunTask(language proto.Language, constraints limits, metaFile string, consoleInput bool) (*box.MetaFile, error) {
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
		b.Box.Config.InputFile = "/box/input.in"
		b.Box.Config.OutputFile = "/box/input.out"
	}

	defer func() {
		b.Box.Config = oldConf
	}()

	_, _, err := b.Box.ExecCommand(language.RunCommand...)
	if metaFile != "" {
		data, err := ioutil.ReadFile(metaFile)
		if err != nil {
			return nil, err
		}
		return box.ParseMetaFile(string(data)), nil
	}
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ExecuteTest executes a new test
func (b *BoxManager) ExecuteSTask(task proto.STask) (*proto.STResponse, error) {
	defer func() {
		// After doing stuff, we need to clean up after ourselves ;)
		if err := b.Reset(); err != nil {
			fmt.Printf("CAN'T RESET BOX %d: %d", b.ID, err)
		}
	}()

	response := &proto.STResponse{TID: task.TID}

	if err := b.Box.WriteFile("/box/"+task.Filename+".in", []byte(task.Input)); err != nil {
		fmt.Println("Can't write input file:", err)
		response.Comments = "Sandbox error: Couldn't write input file"
		return response, err
	}
	consoleInput := task.Filename == "input"

	lang := proto.Languages[task.Language]
	/*if lang.IsCompiled {*/
	if err := b.Box.CopyInBox(path.Join(compilePath, fmt.Sprintf("%d.bin", task.ID)), lang.CompiledName); err != nil {
		response.Comments = "Couldn't link executable in box"
		return response, err
	}
	/* TODO: Change SourceName in interpreted languages to CompiledName (if I haven't done this already)
	} else {
		if err := b.Box.WriteFile(lang.SourceName, []byte(task.SourceCode)); err != nil {
			response.Comments = "Couldn't write interpreter file"
			return response, err
		}
	}*/

	lim := limits{
		MemoryLimit: task.MemoryLimit,
		StackLimit:  task.StackLimit,
		TimeLimit:   task.TimeLimit,
	}
	meta, err := b.RunTask(proto.Languages[task.Language], lim, strconv.Itoa(int(task.ID))+".txt", consoleInput)
	response.Time = meta.Time
	response.Memory = meta.CgMem

	if err != nil {
		response.Comments = fmt.Sprintf("Error running task: %v", err)
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

	file, err := b.Box.GetFile("/box/" + task.Filename + ".out")
	if err != nil {
		response.Comments = "Missing output file"
		return response, nil
	}
	response.Output = string(file)

	return response, nil
	/*
		TODO: MOVE THIS PART TO THE CLIENT SIDE, SINCE IT'S SAFER TO NOT HAVE THE OUTPUT

		// Checking if files are ok
		taskOut, err := b.Box.GetFile("/box/" + task.Problem.TestName + ".out")
		if err != nil {
			if os.IsNotExist(err) {
				b.UpdateChan <- testOutputUpdate{
					id:     test.ID,
					output: "Missing output file",
					score:  0,
				}
				return 0, err
			}
			fmt.Println("Some error happened and idk what to do:", err)
			b.UpdateChan <- testOutputUpdate{
				id:     test.ID,
				output: "Internal grader error",
				score:  -5,
			}
			return -5, err
		}

		tOut = bytes.TrimSpace(tOut)
		tOut = bytes.ReplaceAll(tOut, []byte{'\r', '\n'}, []byte{'\n'})
		taskOut = bytes.TrimSpace(taskOut)
		taskOut = bytes.ReplaceAll(taskOut, []byte{'\r', '\n'}, []byte{'\n'})

		if bytes.Equal(tOut, taskOut) {
			testScore = test.Test.Score
			b.UpdateChan <- testOutputUpdate{
				id:     test.ID,
				output: "Correct",
				score:  test.Test.Score,
			}
		} else {
			testScore = 0
			b.UpdateChan <- testOutputUpdate{
				id:     test.ID,
				output: "Wrong Answer",
				score:  0,
			}
		}
	*/
}

func (b *BoxManager) CompileTask(c proto.Compile) *proto.CResponse {
	defer b.Reset()
	lang := proto.Languages[c.Language]

	outName := path.Join(compilePath, fmt.Sprintf("%d.bin", c.ID))
	resp := &proto.CResponse{}
	resp.Success = true
	resp.ID = c.ID

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

		return resp
	}

	// TODO: Run syntax checker for interpreted languages
	err := ioutil.WriteFile(outName, []byte(c.Code), 0644)
	if err != nil {
		resp.Other = err.Error()
		resp.Success = false
	}

	return resp
}

/*
// ExecuteTask executes a specific task
func (b *BoxManager) ExecuteTask(task common.Task) error {
	// move the compiled binary to tmp if necessary
	lang := common.Languages[task.Language]
	if lang.IsCompiled {
		if err := b.Box.MoveFromBox(lang.CompiledName, path.Join("/tmp", fmt.Sprintf("%d.bin", task.ID))); err != nil {
			return err
		}
	}

	if err := b.Reset(); err != nil {
		return err
	}

	return nil
}*/

// Reset reintializes a box
// Should be run after finishing running a batch of tests
func (b *BoxManager) Reset() (err error) {
	err = b.Box.Cleanup()
	if err != nil {
		return
	}
	b.Box, err = box.NewBox(box.Config{ID: b.ID, Cgroups: true})
	b.Box.Debug = b.debug
	return
}

func SetCompilePath(path string) {
	compilePath = path
}

// New creates a new box manager
func New(id int) (*BoxManager, error) {
	b, err := box.NewBox(box.Config{ID: id, Cgroups: true})
	if err != nil {
		return nil, err
	}
	b.Config.EnvToSet = make(map[string]string)

	bm := &BoxManager{
		ID:  id,
		Box: b,
	}
	return bm, nil
}
