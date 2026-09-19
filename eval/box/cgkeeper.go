package box

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/KiloProjects/kilonova/domain/config"
)

var (
	isolatePath = ""

	keeperMu   sync.Mutex
	keeperDone bool
	keeperErr  error
)

// keeperInit runs InitKeeper once per process and replays its result to every
// later caller, so a probe and a real run agree on whether the sandbox works.
func keeperInit(ctx context.Context) error {
	keeperMu.Lock()
	defer keeperMu.Unlock()
	if keeperDone {
		return keeperErr
	}
	keeperDone = true
	keeperErr = InitKeeper(ctx)
	return keeperErr
}

const (
	cgroupFS = "/sys/fs/cgroup"

	// Isolate's compiled-in default when the config file says nothing.
	defaultCgRoot = "auto:/run/isolate/cgroup"

	// Leaf we park ourselves in so the parent can carry controllers for the
	// per-box cgroups isolate creates.
	daemonLeaf = "daemon"
)

// isolateConfigPaths mirrors the sysconfdir choices isolate is usually built with.
var isolateConfigPaths = []string{"/usr/local/etc/isolate", "/etc/isolate"}

func InitKeeper(ctx context.Context) error {
	if err := initIsolatePath(ctx); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Initialized sandbox binary path", slog.String("path", isolatePath))

	if !config.Eval.EnsureCGKeeper {
		return nil
	}

	return setupCgroup(ctx)
}

// setupCgroup does what isolate's `isolate-cg-keeper` does under systemd, so a
// container needs no init system: it claims the cgroup this process is already
// in, publishes it where isolate looks, and enables the controllers isolate
// needs on the per-box subgroups it will create.
//
// Upstream needs a separate daemon only because a systemd unit requires some
// process to hold its delegated cgroup open; kn grader-serve is already that
// long-lived process, so this sets up and returns.
// Reference: https://github.com/ioi/isolate/blob/master/isolate-cg-keeper.c
func setupCgroup(ctx context.Context) error {
	if err := checkCgroupV2(cgroupFS); err != nil {
		return err
	}

	cgRoot, err := readCgRoot(isolateConfigPaths)
	if err != nil {
		return err
	}

	cg := cgRoot
	if autoFile, ok := strings.CutPrefix(cgRoot, "auto:"); ok {
		procCgroup, err := os.ReadFile("/proc/self/cgroup")
		if err != nil {
			return fmt.Errorf("read /proc/self/cgroup: %w", err)
		}
		cg, err = ownCgroup(string(procCgroup), cgroupFS)
		if err != nil {
			return err
		}
		// A restart in place starts us inside the leaf a previous run moved into.
		// Without this we would nest a second leaf and then fail to enable
		// controllers on a cgroup that still holds the old process.
		if filepath.Base(cg) == daemonLeaf {
			cg = filepath.Dir(cg)
		}
		// isolate reads this file to find the same root we just resolved.
		if err := os.MkdirAll(filepath.Dir(autoFile), 0o755); err != nil {
			return fmt.Errorf("create dir for %s: %w", autoFile, err)
		}
		if err := os.WriteFile(autoFile, []byte(cg+"\n"), 0o644); err != nil {
			return fmt.Errorf("publish cgroup root to %s: %w", autoFile, err)
		}
	}

	if _, err := os.Stat(cg); err != nil {
		return fmt.Errorf("control group root %s does not exist: %w", cg, err)
	}

	// cgroup v2 forbids a cgroup from both holding processes and enabling
	// controllers for its children, so we move ourselves into a leaf first.
	// Unlike upstream we tolerate an existing subgroup: restarting inside a
	// container whose cgroup outlives the process is normal, not an error.
	daemonCg := filepath.Join(cg, daemonLeaf)
	if err := os.MkdirAll(daemonCg, 0o777); err != nil {
		return fmt.Errorf("create subgroup %s: %w", daemonCg, err)
	}
	if err := os.WriteFile(filepath.Join(daemonCg, "cgroup.procs"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		return fmt.Errorf("move self into %s: %w", daemonCg, err)
	}
	if err := os.WriteFile(filepath.Join(cg, "cgroup.subtree_control"), []byte("+cpuset +memory\n"), 0o644); err != nil {
		return fmt.Errorf("enable controllers on %s: %w", cg, err)
	}

	slog.InfoContext(ctx, "Prepared sandbox control group", slog.String("cgroup", cg), slog.String("self", daemonCg))
	return nil
}

// checkCgroupV2 rejects the cgroup layouts isolate cannot use.
func checkCgroupV2(fsRoot string) error {
	if _, err := os.Stat(fsRoot); err != nil {
		return fmt.Errorf("cannot find %s: %w", fsRoot, err)
	}
	if _, err := os.Stat(filepath.Join(fsRoot, "unified")); err == nil {
		return errors.New("combined cgroup v1+v2 mode is not supported")
	}
	if _, err := os.Stat(filepath.Join(fsRoot, "cgroup.subtree_control")); err != nil {
		return fmt.Errorf("cgroup v2 not found at %s: %w", fsRoot, err)
	}
	return nil
}

// readCgRoot returns the cg_root setting from the first isolate config that
// exists, or isolate's default when no config names it.
func readCgRoot(paths []string) (string, error) {
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return "", fmt.Errorf("read isolate config %s: %w", p, err)
		}
		if root := parseCgRoot(string(data)); root != "" {
			return root, nil
		}
		return defaultCgRoot, nil
	}
	return defaultCgRoot, nil
}

// parseCgRoot picks the cg_root value out of an isolate config file, ignoring
// comments. Returns "" when the key is absent.
func parseCgRoot(contents string) string {
	for line := range strings.SplitSeq(contents, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, found := strings.Cut(line, "=")
		if !found || strings.TrimSpace(key) != "cg_root" {
			continue
		}
		return strings.TrimSpace(val)
	}
	return ""
}

// ownCgroup resolves this process's cgroup from the contents of
// /proc/self/cgroup. The "0::<path>" line is the unified hierarchy. Joining it
// onto the cgroup mount is correct in both namespace modes: with a host cgroup
// namespace the path is absolute on the host and the mount is the host root,
// and with a private one the path is "/" and the mount is already our own group.
func ownCgroup(procSelfCgroup string, fsRoot string) (string, error) {
	for line := range strings.SplitSeq(procSelfCgroup, "\n") {
		if path, ok := strings.CutPrefix(strings.TrimSpace(line), "0::"); ok {
			return filepath.Join(fsRoot, path), nil
		}
	}
	return "", errors.New("cannot find my own cgroup in /proc/self/cgroup")
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
