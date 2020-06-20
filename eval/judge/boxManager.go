package judge

import (
	"context"
	"fmt"

	"github.com/AlexVasiluta/kilonova/eval/box"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
)

// BoxManager manages a box with eval-based tasks
type BoxManager struct {
	ID       int
	Box      *box.Box
	TaskChan chan models.EvalTest

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
func (b *BoxManager) CompileFile(SourceCode string, language models.Language) error {
	if err := b.Box.WriteFile(language.SourceName, SourceCode); err != nil {
		return err
	}

	// If the language is not compiled, we don't need to compile it
	if !language.IsCompiled {
		return nil
	}

	oldConfig := b.Box.Config
	b.Box.Config.InheritEnv = true
	for _, dir := range language.Mounts {
		b.Box.Config.Directories = append(b.Box.Config.Directories, dir)
	}

	_, _, err := b.Box.ExecCommand(language.CompileCommand...)
	b.Box.Config = oldConfig
	return err
}

// RunTask runs a program, following the language conventions
func (b *BoxManager) RunTask(language models.Language, constraints models.Limits) error {
	oldConf := b.Box.Config

	// if our specified language is not compiled, then it means that
	// the mounts specified should be added at runtime
	if !language.IsCompiled {
		for _, dir := range language.Mounts {
			b.Box.Config.Directories = append(b.Box.Config.Directories, dir)
		}
	}

	b.Box.Config.MemoryLimit = constraints.MemoryLimit
	b.Box.Config.StackSize = constraints.StackLimit
	b.Box.Config.TimeLimit = constraints.TimeLimit
	b.Box.Config.WallTimeLimit = constraints.TimeLimit + 0.5

	so, se, err := b.Box.ExecCommand(language.RunCommand...)

	b.Box.Config = oldConf

	// Debug output
	if b.debug {
		fmt.Println("-SO-")
		fmt.Println(so)
		fmt.Println("-SE-")
		fmt.Println(se)
		fmt.Println("-ER-")
		fmt.Println(err)
	}

	fmt.Println("Program output: ", so)

	return err
}

// CleanupTask removes all task testing-specific items (ie test files)
func (b *BoxManager) CleanupTask(files ...string) error {
	for _, file := range files {
		if err := b.Box.RemoveFile(file); err != nil {
			return err
		}
	}
	return nil
}

// Start returns a channel to send tasks to
func (b *BoxManager) Start(ctx context.Context) chan models.EvalTest {
	b.TaskChan = make(chan models.EvalTest)
	go func() {
		for {
			select {
			case task := <-b.TaskChan:
				b.CompileFile(task.Task.SourceCode, models.Languages["cpp"])
				spew.Dump(task)
			case <-ctx.Done():
				fmt.Println("Ending box manager")
				b.Box.Cleanup()
				return
			}
		}
	}()
	return b.TaskChan
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

// NewBoxManager creates a new box
func NewBoxManager(id int) (*BoxManager, error) {
	b, err := box.NewBox(box.Config{ID: id})
	if err != nil {
		return nil, err
	}
	bm := &BoxManager{ID: id, Box: b}
	return bm, nil
}
