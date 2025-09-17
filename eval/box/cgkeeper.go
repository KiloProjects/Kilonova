package box

import (
	"context"
	"errors"
	"log/slog"
	"os/exec"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

var (
	isolatePath = ""
)

// func verifyKeeper() error {
// 	panic("TODO")
// }

// func startKeeper() error {
// 	panic("TODO")
// }

func InitKeeper(ctx context.Context) error {
	if err := initIsolatePath(ctx); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Initialized sandbox binary path", slog.String("path", isolatePath))

	if !flags.EnsureCGKeeper.Value() {
		return nil
	}

	panic("TODO")
}

func initIsolatePath(ctx context.Context) error {
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
	slog.ErrorContext(ctx, "Sandbox binary not found. Set it up using scripts/init_isolate_cg2.sh")
	return errors.New("no isolate binary found")
}
