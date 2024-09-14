package eval

import (
	"os"
	"os/exec"
	"path/filepath"

	"go.uber.org/zap"
)

func disableLang(key string) {
	lang := Langs[key]
	lang.Disabled = true
	Langs[key] = lang
}

// checkLanguages disables all languages that are *not* detected by the system in the current configuration
// It should be run at the start of the execution (and implemented more nicely tbh)
// TODO: do it on scheduler initialization
func checkLanguages() {
	for k, v := range Langs {
		if v.Disabled { // Skip search if already disabled
			continue
		}
		var toSearch []string
		if v.Compiled {
			toSearch = v.CompileCommand
		} else {
			toSearch = v.RunCommand
		}
		if len(toSearch) == 0 {
			disableLang(k)
			zap.S().Infof("Language %q was disabled because of empty line", k)
			continue
		}
		cmd, err := exec.LookPath(toSearch[0])
		if err != nil {
			disableLang(k)
			zap.S().Infof("Language %q was disabled because the compiler/interpreter was not found in PATH", k)
			continue
		}
		cmd, err = filepath.EvalSymlinks(cmd)
		if err != nil {
			disableLang(k)
			zap.S().Infof("Language %q was disabled because the compiler/interpreter had a bad symlink", k)
			continue
		}

		// TODO: exec.LookPath also checks if the file has execute perms, do we need to do this?
		stat, err := os.Stat(cmd)
		if err != nil {
			disableLang(k)
			zap.S().Infof("Language %q was disabled because the compiler/interpreter binary was not found", k)
			continue
		}

		if stat.Mode()&0111 == 0 {
			disableLang(k)
			zap.S().Infof("Language %q was disabled because the compiler/interpreter binary is not executable", k)
		}

	}
}

// Initialize should be called after reading the flags, but before manager.New
func Initialize() error {
	checkLanguages()
	return nil
}
