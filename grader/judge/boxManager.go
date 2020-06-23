package judge

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/grader/box"
	"github.com/KiloProjects/Kilonova/models"
)

// BoxManager manages a box with eval-based tasks
type BoxManager struct {
	ID         int
	Box        *box.Box
	TaskChan   chan models.Task
	UpdateChan chan models.Updater

	DataManager *datamanager.Manager

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
func (b *BoxManager) CompileFile(SourceCode string, language models.Language) (string, error) {
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
func (b *BoxManager) RunTask(language models.Language, constraints models.Limits, metaFile string, problemFile string) error {
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

	b.Box.Config.MemoryLimit = constraints.MemoryLimit
	b.Box.Config.StackSize = constraints.StackLimit
	b.Box.Config.TimeLimit = constraints.TimeLimit
	b.Box.Config.WallTimeLimit = constraints.TimeLimit + 0.5

	if metaFile != "" {
		b.Box.Config.MetaFile = "/tmp/" + metaFile
	}

	if problemFile != "" {
		b.Box.Config.InputFile = "/box/" + problemFile + ".in"
		b.Box.Config.OutputFile = "/box/" + problemFile + ".out"
	}

	_, _, err := b.Box.ExecCommand(language.RunCommand...)

	b.Box.Config = oldConf

	return err
}

// Start returns a channel to send tasks to
func (b *BoxManager) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case task := <-b.TaskChan:
				fmt.Println("Running task", task.ID)

				// TODO: This is not that barebones but needs more testing, HANDLE IMPERFECT CASES

				b.UpdateChan <- taskStatusUpdate{id: task.ID, status: models.StatusWorking}

				// Compile once for the compile output
				compileOut, err := b.CompileFile(task.SourceCode, models.Languages[task.Language])
				compileOut = strings.TrimSpace(compileOut)
				if err != nil {
					b.UpdateChan <- taskCompileUpdate{id: task.ID, compileMessage: compileOut, isFatal: true}
					b.UpdateChan <- taskStatusUpdate{id: task.ID, status: models.StatusDone}

					if err := b.Reset(); err != nil {
						fmt.Println("DAFUQ, CAN'T RESET: ", err)
					}
					continue
				}
				b.UpdateChan <- taskCompileUpdate{id: task.ID, compileMessage: compileOut, isFatal: false}
				var score int
				for _, test := range task.Tests {

					tIn, tOut, err := b.DataManager.GetTest(task.ProblemID, test.TestID)
					if err != nil {
						fmt.Println("Can't get tests:", err)
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: "Internal grader error",
							score:  -8,
						}
						if err := b.Reset(); err != nil {
							fmt.Println("DAFUQ, CAN'T RESET: ", err)
						}
						continue
					}

					if err := b.Box.WriteFile("/box/"+task.Problem.TestName+".in", tIn); err != nil {
						fmt.Println("Can't write input file:", err)
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: "Internal grader error",
							score:  -7,
						}
						if err := b.Reset(); err != nil {
							fmt.Println("DAFUQ, CAN'T RESET: ", err)
						}
						continue
					}

					// FIXME(alexv): This is very very inefficient for languages like C++, this must be fixed before the beta
					if _, err := b.CompileFile(task.SourceCode, models.Languages[task.Language]); err != nil {
						// This should never happen unless something actually got messed up
						fmt.Println("(DAFUQ) Error compiling file **IN TEST**:", err)
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: "Internal grader error",
							score:  -6,
						}
						if err := b.Reset(); err != nil {
							fmt.Println("DAFUQ, CAN'T RESET: ", err)
						}
						continue
					}

					var testName string
					if task.Problem.ConsoleInput {
						testName = task.Problem.TestName
					}
					if err := b.RunTask(models.Languages[task.Language], task.Problem.Limits, strconv.Itoa(int(task.ID))+".txt", testName); err != nil {
						fmt.Println("Error running task:", err)
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: err.Error(),
							score:  0,
						}
						if err := b.Reset(); err != nil {
							fmt.Println("DAFUQ, CAN'T RESET: ", err)
						}
						continue
						// continue
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
						} else {
							fmt.Println("Some error happened and idk what to do:", err)
							b.UpdateChan <- testOutputUpdate{
								id:     test.ID,
								output: "Internal grader error",
								score:  -5,
							}
						}
						if err := b.Reset(); err != nil {
							fmt.Println("DAFUQ, CAN'T RESET: ", err)
						}
						continue
					}
					tOut = strings.TrimSpace(tOut)
					tOut = strings.ReplaceAll(tOut, "\r\n", "\n")
					taskOut = bytes.TrimSpace(taskOut)
					taskOut = bytes.ReplaceAll(taskOut, []byte{'\r', '\n'}, []byte{'\n'})
					if tOut == string(taskOut) {
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: "Correct",
							score:  test.Test.Score,
						}
						score += test.Test.Score
					} else {
						b.UpdateChan <- testOutputUpdate{
							id:     test.ID,
							output: "Wrong Answer",
							score:  0,
						}
					}

					// After doing stuff, we need to clean up after ourselves ;)
					if err := b.Reset(); err != nil {
						fmt.Println("DAFUQ, CAN'T RESET: ", err)
					}

				}
				b.UpdateChan <- taskScoreUpdate{id: task.ID, score: score}
				b.UpdateChan <- taskStatusUpdate{id: task.ID, status: models.StatusDone}
				b.Reset()
				fmt.Println()
				fmt.Println()
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
	b.Box, err = box.NewBox(box.Config{ID: b.ID})
	b.Box.Debug = b.debug
	return
}

// NewBoxManager creates a new box manager
func NewBoxManager(id int, dataManager *datamanager.Manager, TaskChan chan models.Task, UpdateChan chan models.Updater) (*BoxManager, error) {
	b, err := box.NewBox(box.Config{ID: id})
	if err != nil {
		return nil, err
	}
	b.Config.EnvToSet = make(map[string]string)
	if TaskChan == nil {
		TaskChan = make(chan models.Task, 4)
	}
	if UpdateChan == nil {
		UpdateChan = make(chan models.Updater, 10)
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
