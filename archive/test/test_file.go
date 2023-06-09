package test

import (
	"archive/zip"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/KiloProjects/kilonova"
)

var (
	ErrInvalidTestID = kilonova.Statusf(400, "Invalid test ID")
)

func getTestID(name string) (int, *kilonova.StatusError) {
	var tid int
	if _, err := fmt.Sscanf(name, "%d-", &tid); err != nil {
		// maybe it's problem_name.%d.{in,sol,out} format
		nm := strings.Split(strings.TrimSuffix(name, path.Ext(name)), ".")
		if len(nm) == 0 {
			return -1, ErrInvalidTestID
		}
		val, err := strconv.Atoi(nm[len(nm)-1])
		if err != nil {
			return -1, ErrInvalidTestID
		}
		return val, nil
	}
	return tid, nil
}

func ProcessTestInputFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	tid, err := getTestID(path.Base(file.Name))
	if err != nil {
		return nil
	}
	tf := ctx.tests[tid]
	if tf.InFile != nil { // in file already exists
		return kilonova.Statusf(400, "Multiple input files for test %d", tid)
	}

	tf.InFile = file
	ctx.tests[tid] = tf
	return nil
}

func ProcessTestOutputFile(ctx *ArchiveCtx, file *zip.File) *kilonova.StatusError {
	tid, err := getTestID(path.Base(file.Name))
	if err != nil {
		return nil
	}
	tf := ctx.tests[tid]
	if tf.OutFile != nil { // out file already exists
		return kilonova.Statusf(400, "Multiple output files for test %d", tid)
	}

	tf.OutFile = file
	ctx.tests[tid] = tf
	return nil
}
