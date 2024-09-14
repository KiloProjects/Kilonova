package box

import (
	"errors"
	"log/slog"
	"os/exec"

	"github.com/KiloProjects/kilonova/internal/config"
)

var (
	EnsureCGKeeper    = config.GenFlag("feature.grader.ensure_keeper", false, "Ensure isolate-cg-keeper is running")
	IsolateConfigPath = config.GenFlag("feature.grader.isolate_config_path", "/usr/local/etc/isolate", "Configuration path for isolate sandbox")

	isolatePath = ""
)

// func verifyKeeper() error {
// 	panic("TODO")
// }

// func startKeeper() error {
// 	panic("TODO")
// }

func InitKeeper() error {
	if err := initIsolatePath(); err != nil {
		return err
	}

	slog.Info("Initialized sandbox binary path", slog.String("path", isolatePath))

	if !EnsureCGKeeper.Value() {
		return nil
	}

	panic("TODO")
}

func initIsolatePath() error {
	for _, path := range []string{
		"/usr/local/bin/isolate",     // Official path
		"/usr/local/etc/isolate_bin", // Cgroup v1 path
		"isolate",                    // Lookup in other path
	} {
		p, err := exec.LookPath(path)
		if err == nil {
			isolatePath = p
			return nil
		}
	}
	slog.Error("Sandbox binary not found. Set it up using scripts/init_isolate_cg2.sh")
	return errors.New("no isolate binary found")
}
