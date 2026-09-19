package box

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseCgRoot(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     string
	}{
		{"auto form", "box_root = /var/local/lib/isolate\ncg_root = auto:/run/isolate/cgroup\n", "auto:/run/isolate/cgroup"},
		{"explicit path", "cg_root = /sys/fs/cgroup/isolate.slice/isolate.service\n", "/sys/fs/cgroup/isolate.slice/isolate.service"},
		{"commented out", "# cg_root = /sys/fs/cgroup/nope\nbox_root = /box\n", ""},
		{"absent", "box_root = /box\nlock_root = /run/isolate/locks\n", ""},
		{"extra whitespace", "   cg_root   =   auto:/tmp/cg   \n", "auto:/tmp/cg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parseCgRoot(tt.contents); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

// A missing config must not be an error: isolate has a compiled-in default.
func TestReadCgRootFallsBackToDefault(t *testing.T) {
	got, err := readCgRoot([]string{filepath.Join(t.TempDir(), "absent")})
	if err != nil {
		t.Fatal(err)
	}
	if got != defaultCgRoot {
		t.Fatalf("got %q, want %q", got, defaultCgRoot)
	}
}

func TestReadCgRootPicksFirstExisting(t *testing.T) {
	dir := t.TempDir()
	second := filepath.Join(dir, "isolate")
	if err := os.WriteFile(second, []byte("cg_root = /sys/fs/cgroup/custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readCgRoot([]string{filepath.Join(dir, "absent"), second})
	if err != nil {
		t.Fatal(err)
	}
	if got != "/sys/fs/cgroup/custom" {
		t.Fatalf("got %q", got)
	}
}

// Both cgroup namespace modes have to resolve to a real path: host mode gives an
// absolute host path, private mode gives "/".
func TestOwnCgroup(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     string
	}{
		{
			"host cgroup namespace",
			"0::/system.slice/docker-abc123.scope\n",
			"/sys/fs/cgroup/system.slice/docker-abc123.scope",
		},
		{
			"private cgroup namespace",
			"0::/\n",
			"/sys/fs/cgroup",
		},
		{
			"v1 lines ignored",
			"12:pids:/user.slice\n0::/user.slice/session-3.scope\n",
			"/sys/fs/cgroup/user.slice/session-3.scope",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ownCgroup(tt.contents, "/sys/fs/cgroup")
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOwnCgroupMissingUnifiedLine(t *testing.T) {
	if _, err := ownCgroup("12:pids:/user.slice\n", "/sys/fs/cgroup"); err == nil {
		t.Fatal("expected an error when there is no 0:: line")
	}
}

func TestCheckCgroupV2(t *testing.T) {
	t.Run("rejects hybrid", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, "unified"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := checkCgroupV2(dir); err == nil {
			t.Fatal("hybrid cgroup layout must be rejected")
		}
	})

	t.Run("rejects v1", func(t *testing.T) {
		if err := checkCgroupV2(t.TempDir()); err == nil {
			t.Fatal("missing cgroup.subtree_control must be rejected")
		}
	})

	t.Run("accepts v2", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "cgroup.subtree_control"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := checkCgroupV2(dir); err != nil {
			t.Fatalf("valid v2 layout rejected: %v", err)
		}
	})
}
