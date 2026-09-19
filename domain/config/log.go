package config

import (
	"io"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

// logToFile decides where the rotating side-channel logs go. Default true keeps
// host deployments writing <KN_DATA_DIR>/logs; containers set KN_LOG_FILE=false.
var logToFile = true

// SetLogToFile is called once at startup, before any logger is built.
func SetLogToFile(v bool) { logToFile = v }

// LogToFile reports whether rotating log files are enabled, for callers that
// need to skip creating the directory.
func LogToFile() bool { return logToFile }

// LogWriter returns the rotating file named under LogDir, or stdout when file
// logging is disabled. Stdout rather than io.Discard on purpose: these are
// dedicated logs (db, grader, sandbox runs, ...) whose content goes nowhere
// else, and in a container the only sink there is is the process output.
func LogWriter(name string, maxSizeMB int) io.Writer {
	if !logToFile {
		return os.Stdout
	}
	return &lumberjack.Logger{
		Filename: filepath.Join(Common.LogDir(), name),
		MaxSize:  maxSizeMB,
		Compress: true,
	}
}
