package eval

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

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

	// Test right now if they exist
	zap.S().Info("Isolate path: ", config.Eval.IsolatePath)
	if _, err := os.Stat(config.Eval.IsolatePath); os.IsNotExist(err) {
		zap.S().Fatal("Sandbox binary not found. Run scripts/init_isolate.sh to properly install it.")
	}

	if err := os.MkdirAll(config.Eval.CompilePath, 0777); err != nil {
		return err
	}

	checkLanguages()

	return nil
}

// // Initialize should be called after reading the flags, but before manager.New
// func Initialize() error {

// 	// Test right now if they exist
// 	if _, err := os.Stat(config.Eval.IsolatePath); os.IsNotExist(err) {
// 		// download isolate
// 		fmt.Println("Downloading isolate binary")
// 		if err := downloadFile(isolateURL, config.Eval.IsolatePath, 0744); err != nil {
// 			return err
// 		}
// 		fmt.Println("Isolate binary downloaded")
// 	}
// 	if _, err := os.Stat(config.Eval.IsolatePath); os.IsNotExist(err) {
// 		// download the config file
// 		fmt.Println("Downloading isolate config")
// 		if err := downloadFile(configURL, config.Eval.IsolatePath, 0644); err != nil {
// 			return err
// 		}
// 		fmt.Println("Isolate config downloaded")
// 	}

// 	if err := os.MkdirAll(config.Eval.CompilePath, 0777); err != nil {
// 		return err
// 	}

// 	checkLanguages()

// 	return nil
// }

// func downloadFile(url, path string, perm os.FileMode) error {
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return err
// 	}

// 	file, err := os.Create(path)
// 	if err != nil {
// 		return err
// 	}
// 	defer file.Close()

// 	_, err = io.Copy(file, resp.Body)
// 	if err != nil {
// 		return err
// 	}

// 	return file.Chmod(perm)
// }
