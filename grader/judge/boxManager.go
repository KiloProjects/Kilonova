package judge

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader/box"
	"github.com/davecgh/go-spew/spew"
)

// BoxManager manages a box with eval-based tasks
type BoxManager struct {
	ID         int
	Box        *box.Box
	TaskChan   chan common.Task
	UpdateChan chan common.Updater

	DataManager datamanager.Manager

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
func (b *BoxManager) CompileFile(SourceCode []byte, language common.Language) (string, error) {
	if err := b.Box.WriteFile(language.SourceName, SourceCode); err != nil {
		return "", err
	}

	// If the language is not compiled, we don't need to compile it
	if !language.IsCompiled {
		return "", nil
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

// RunTask runs a program, following the language conventions
// filenames contains the names for input and output, used if consoleInput is true
func (b *BoxManager) RunTask(language common.Language, constraints common.Limits, metaFile string, problemFile string) (*box.MetaFile, error) {
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

	// unfortunately, the memory limits can't have a floating point number of kbytes
	// so we have to multiply by 1024 (to get the number of kbytes) and truncate the number
	b.Box.Config.MemoryLimit = int(constraints.MemoryLimit * 1024)
	// CgroupMem is disabled for now, it causes a sandbox error "Cannot set /sys/fs/cgroup/memory/box-2/memory.limit_in_bytes"
	// and i don't want to deal with it right now
	// b.Box.Config.CgroupMem = int(constraints.MemoryLimit * 1024)
	b.Box.Config.StackSize = int(constraints.StackLimit * 1024)
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

	if problemFile != "" {
		b.Box.Config.InputFile = path.Join("/box/", problemFile+".in")
		b.Box.Config.OutputFile = path.Join("/box/", problemFile+".out")
	}

	_, _, err := b.Box.ExecCommand(language.RunCommand...)
	if metaFile != "" {
		data, err := ioutil.ReadFile(metaFile)
		if err != nil {
			return nil, err
		}
		return box.ParseMetaFile(string(data)), nil
	}

	b.Box.Config = oldConf
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ExecuteTest executes a new test
func (b *BoxManager) ExecuteTest(task common.Task, test common.EvalTest) (int, error) {
	var testScore int
	defer func() {
		// After doing stuff, we need to clean up after ourselves ;)
		if err := b.Reset(); err != nil {
			fmt.Printf("CAN'T RESET BOX %d: %d", b.ID, err)
		}
	}()
	tIn, tOut, err := b.DataManager.GetTest(task.ProblemID, test.Test.VisibleID)
	if err != nil {
		fmt.Println("Can't get tests:", err)
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: "Internal grader error",
			score:  -8,
		}
		return -8, err
	}

	if err := b.Box.WriteFile("/box/"+task.Problem.TestName+".in", tIn); err != nil {
		fmt.Println("Can't write input file:", err)
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: "Internal grader error",
			score:  -7,
		}
		return -7, err
	}

	lang := common.Languages[task.Language]
	if lang.IsCompiled {
		if err := b.Box.LinkInBox(path.Join("/tmp", fmt.Sprintf("%d.bin", task.ID)), lang.CompiledName); err != nil {
			b.UpdateChan <- testOutputUpdate{
				id:     test.ID,
				output: "Internal grader error",
				score:  -6,
			}
			return -6, err
		}
	} else {
		if err := b.Box.WriteFile(lang.SourceName, []byte(task.SourceCode)); err != nil {
			b.UpdateChan <- testOutputUpdate{
				id:     test.ID,
				output: "Internal grader error",
				score:  -6,
			}
			return -6, err
		}
	}

	var testName string
	if task.Problem.ConsoleInput {
		testName = task.Problem.TestName
	}
	meta, err := b.RunTask(common.Languages[task.Language], task.Problem.Limits, strconv.Itoa(int(task.ID))+".txt", testName)
	spew.Dump(meta)
	b.UpdateChan <- testMetaUpdate{id: test.ID, meta: meta}
	if meta.Status == "TO" { // Time limit exceeded
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: "Time limit exceeded",
			score:  0,
		}
		return 0, nil
	}
	if meta.Status == "RE" { // Runtime Error
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: "Runtime error",
			score:  0,
		}
		return 0, nil
	}
	if meta.Status == "SG" { // Program died on a signal
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: meta.Message,
			score:  0,
		}
		return 0, nil
	}
	if meta.Status == "XX" { // Sandbox error
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: "Sandbox error",
			score:  0,
		}
		return 0, nil
	}
	if err != nil {
		fmt.Println("Error running task:", err)
		b.UpdateChan <- testOutputUpdate{
			id:     test.ID,
			output: err.Error(),
			score:  0,
		}
		return 0, err
	}

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

	return testScore, nil
}

// ExecuteTask executes a specific task
func (b *BoxManager) ExecuteTask(task common.Task) error {
	// TODO: this might be slightly buggy, fix it

	b.UpdateChan <- taskStatusUpdate{id: task.ID, status: common.StatusWorking}

	// Compile once for the compile output
	compileOut, err := b.CompileFile([]byte(task.SourceCode), common.Languages[task.Language])
	compileOut = strings.TrimSpace(compileOut)
	if err != nil {
		b.UpdateChan <- taskCompileUpdate{id: task.ID, compileMessage: compileOut, isFatal: true}
		b.UpdateChan <- taskStatusUpdate{id: task.ID, status: common.StatusDone}
		return nil
	}
	b.UpdateChan <- taskCompileUpdate{id: task.ID, compileMessage: compileOut, isFatal: false}

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

	var score int
	for _, test := range task.Tests {
		testScore, err := b.ExecuteTest(task, test)
		if err != nil {
			fmt.Printf("ERROR EXECUTING TEST %d: %s", test.ID, err)
		}
		if testScore > 0 {
			score += testScore
		}
	}
	b.UpdateChan <- taskScoreUpdate{id: task.ID, score: score}
	b.UpdateChan <- taskStatusUpdate{id: task.ID, status: common.StatusDone}
	b.Reset()
	return nil
}

// Start returns a channel to send tasks to
func (b *BoxManager) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case task := <-b.TaskChan:
				fmt.Println("Running task", task.ID)

				if err := b.ExecuteTask(task); err != nil {
					fmt.Println("TASK EXECUTE ERROR:", err)
				}
			case <-ctx.Done():
				fmt.Println("Ending box manager")
				b.Box.Cleanup()
				return
			}
		}
	}()
}

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

// NewBoxManager creates a new box manager
func NewBoxManager(id int, dataManager datamanager.Manager, TaskChan chan common.Task, UpdateChan chan common.Updater) (*BoxManager, error) {
	b, err := box.NewBox(box.Config{ID: id, Cgroups: true})
	if err != nil {
		return nil, err
	}
	b.Config.EnvToSet = make(map[string]string)
	if TaskChan == nil {
		TaskChan = make(chan common.Task, 4)
	}
	if UpdateChan == nil {
		UpdateChan = make(chan common.Updater, 10)
	}

	bm := &BoxManager{
		ID:          id,
		Box:         b,
		DataManager: dataManager,
		TaskChan:    TaskChan,
		UpdateChan:  UpdateChan,
	}
	return bm, nil
}
