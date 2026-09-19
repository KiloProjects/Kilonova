package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

const fixtureConfig = `[common]
 log_dir = "/var/log/kn"
 data_dir = "/var/lib/kn/data"
 debug = false
 host_prefix = "https://kilonova.ro"
 default_language = "ro"
 db_dsn = "sslmode=disable host=/var/run/postgresql dbname=kilonova user=kn password=s3cret application_name=kilonova"
 test_max_mem_kb = 655360
 legacy_unknown = 1

[eval]
 num_concurrent = 4
 global_max_mem_kb = 3145728
 starting_box = 1

[email]
 enabled = false
 host = "smtp:587"
 username = "u"
 password = "p"

[frontend]
 banned_hot_problems = [5, 7]
`

const fixtureGrader = `[grader]
listen = ":9000"
cert_file = "/etc/kilonova/grader.crt"
key_file = "/etc/kilonova/grader.key"
scratch_dir = "/var/lib/kilonova/scratch"
num_concurrent = 4
global_max_mem_kb = 3145728
starting_box = 1

[[grader.client]]
name = "kilonova"
token = "t1"

[[grader.client]]
name = "staging"
token = "t2"
`

func TestMigrateConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config.toml")
	grd := filepath.Join(dir, "grader.toml")
	os.WriteFile(cfg, []byte(fixtureConfig), 0o644)
	os.WriteFile(grd, []byte(fixtureGrader), 0o644)
	before1, _ := os.Stat(cfg)
	before2, _ := os.Stat(grd)

	var out, errOut bytes.Buffer
	if err := migrateConfig(cfg, grd, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	got := strings.Split(strings.TrimSpace(out.String()), "\n")
	want := []string{
		"KN_DATA_DIR=/var/lib/kn/data",
		"KN_HOST_PREFIX=https://kilonova.ro",
		`KN_DB_DSN="sslmode=disable host=/var/run/postgresql dbname=kilonova user=kn password=s3cret application_name=kilonova"`,
		"KN_SANDBOX_NUM_CONCURRENT=4",
		"KN_SANDBOX_GLOBAL_MAX_MEM_KB=3145728",
		"KN_SANDBOX_STARTING_BOX=1",
		"KN_GRADER_LISTEN=:9000",
		"KN_GRADER_TLS_CERT=/etc/kilonova/grader.crt",
		"KN_GRADER_TLS_KEY=/etc/kilonova/grader.key",
		"KN_GRADER_CLIENT_KILONOVA=t1",
		"KN_GRADER_CLIENT_STAGING=t2",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("stdout mismatch:\n got: %q\nwant: %q", got, want)
	}
	if !strings.Contains(errOut.String(), "legacy_unknown") || !strings.Contains(errOut.String(), "log_dir") {

		t.Errorf("unmapped key not reported: %s", errOut.String())
	}

	// Flag-bound values landed in the registry (and therefore flags.json).
	if flags.DefaultLanguage.Value() != "ro" || flags.TestMaxMemKB.Value() != 655360 || !slices.Equal(flags.BannedHotProblems.Value(), []int{5, 7}) {
		t.Fatalf("flags not updated: %q %d %v", flags.DefaultLanguage.Value(), flags.TestMaxMemKB.Value(), flags.BannedHotProblems.Value())
	}
	fp := filepath.Join(dir, "flags.json")
	if err := config.SaveConfigV2(context.Background(), fp); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(fp)
	var saved struct {
		Lang   string `json:"frontend.default_language"`
		Banned []int  `json:"frontend.banned_hot_problems"`
	}
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if saved.Lang != "ro" || !slices.Equal(saved.Banned, []int{5, 7}) {
		t.Fatalf("flags.json: %q %v", saved.Lang, saved.Banned)
	}

	// Inputs untouched; second run identical.
	after1, _ := os.Stat(cfg)
	after2, _ := os.Stat(grd)
	if !after1.ModTime().Equal(before1.ModTime()) || !after2.ModTime().Equal(before2.ModTime()) {
		t.Fatal("source files were modified")
	}
	var again bytes.Buffer
	if err := migrateConfig(cfg, grd, &again, &errOut); err != nil {
		t.Fatal(err)
	}
	if again.String() != out.String() {
		t.Fatal("second run differs")
	}
}

func TestMigrateConfigMissingFile(t *testing.T) {
	dir := t.TempDir()
	grd := filepath.Join(dir, "grader.toml")
	os.WriteFile(grd, []byte(fixtureGrader), 0o644)
	var out, errOut bytes.Buffer
	if err := migrateConfig(filepath.Join(dir, "config.toml"), grd, &out, &errOut); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errOut.String(), "config.toml not found") {
		t.Errorf("missing notice: %s", errOut.String())
	}
	if !strings.Contains(out.String(), "KN_DATA_DIR=/var/lib/kilonova\n") || !strings.Contains(out.String(), "KN_GRADER_CLIENT_KILONOVA=") {

		t.Errorf("unexpected stdout: %s", out.String())
	}
}
