package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"strings"

	"github.com/BurntSushi/toml"
	"github.com/KiloProjects/kilonova/domain/config"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/urfave/cli/v3"
)

// Legacy TOML shapes. This file is the only remaining reader of config.toml
// and grader.toml; the live process uses KN_* environment variables.
type legacyConfig struct {
	Common struct {
		LogDir       string `toml:"log_dir"`
		DataDir      string `toml:"data_dir"`
		Debug        bool   `toml:"debug"`
		HostPrefix   string `toml:"host_prefix"`
		DefaultLang  string `toml:"default_language"`
		DBDSN        string `toml:"db_dsn"`
		TestMaxMemKB int    `toml:"test_max_mem_kb"`
	} `toml:"common"`
	Eval struct {
		Mode          string `toml:"mode"`
		NumConcurrent int    `toml:"num_concurrent"`
		GlobalMaxMem  int64  `toml:"global_max_mem_kb"`
		StartingBox   int    `toml:"starting_box"`
		Remote        struct {
			Endpoint string `toml:"endpoint"`
			Token    string `toml:"token"`
		} `toml:"remote"`
	} `toml:"eval"`
	Email struct {
		Enabled  bool   `toml:"enabled"`
		Host     string `toml:"host"`
		Username string `toml:"username"`
		Password string `toml:"password"`
		SendAs   string `toml:"sendAs"`
	} `toml:"email"`
	Frontend struct {
		BannedHotProblems []int `toml:"banned_hot_problems"`
	} `toml:"frontend"`
}

type legacyGrader struct {
	Grader struct {
		Listen        string `toml:"listen"`
		CertFile      string `toml:"cert_file"`
		KeyFile       string `toml:"key_file"`
		ScratchDir    string `toml:"scratch_dir"`
		NumConcurrent int    `toml:"num_concurrent"`
		GlobalMaxMem  int64  `toml:"global_max_mem_kb"`
		StartingBox   int    `toml:"starting_box"`
		ScratchTTLSec int    `toml:"scratch_ttl_sec"`
		Clients       []struct {
			Name     string `toml:"name"`
			Token    string `toml:"token"`
			Priority string `toml:"priority"`
		} `toml:"client"`
	} `toml:"grader"`
}

var configMigrate = &cli.Command{
	Name:  "config-migrate",
	Usage: "Convert legacy config.toml / grader.toml into KN_* lines (stdout) and flags.json entries; the TOML files are left untouched",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "config", Aliases: []string{"c"}, Usage: "Legacy platform config", Value: "./config.toml"},
		&cli.StringFlag{Name: "grader-config", Usage: "Legacy grader config", Value: "./grader.toml"},
	},
	Action: func(ctx context.Context, command *cli.Command) error {
		return migrateConfig(command.String("config"), command.String("grader-config"), os.Stdout, os.Stderr)
	},
}

// envEmitter collects KEY=value lines in first-seen order, deduplicating keys
// that both files can define (KN_SANDBOX_*).
type envEmitter struct {
	order  []string
	values map[string]string
	errOut io.Writer
}

func (e *envEmitter) set(key string, val any) {
	var s string
	switch v := val.(type) {
	case string:
		s = v
	case bool:
		if !v {
			return
		}
		s = "true"
	case int:
		if v == 0 {
			return
		}
		s = strconv.Itoa(v)
	case int64:
		if v == 0 {
			return
		}
		s = strconv.FormatInt(v, 10)

	default:
		panic(fmt.Sprintf("config-migrate: unsupported value %T", val))
	}
	if s == "" {
		return
	}
	if prev, ok := e.values[key]; ok {
		if prev != s {
			fmt.Fprintf(e.errOut, "warning: %s set by both files (%q, %q); keeping the later one\n", key, prev, s)
		}
	} else {
		e.order = append(e.order, key)
	}
	e.values[key] = s
}

func (e *envEmitter) write(out io.Writer) {
	for _, k := range e.order {
		fmt.Fprintf(out, "%s=%s\n", k, dotenvValue(e.values[k]))
	}
}

// envName turns a legacy client name into an environment variable suffix; the
// grader lowercases it again on read, so only case and punctuation are lost.
func envName(name string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' {
			return r - 'a' + 'A'
		}
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '_'
	}, name)
}

// dotenvValue quotes only when the raw form would not survive a .env parser.
func dotenvValue(s string) string {
	if strings.ContainsAny(s, " \t\n#\"'$\\`") {
		return strconv.Quote(s)
	}
	return s
}

// decodeLegacy returns found=false when the file does not exist.
func decodeLegacy(path string, v any, errOut io.Writer) (found bool, err error) {
	md, err := toml.DecodeFile(path, v)
	if errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(errOut, "notice: %s not found, skipping\n", path)
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("%s: %w", path, err)
	}
	for _, k := range md.Undecoded() {
		fmt.Fprintf(errOut, "warning: %s: key %s has no KN_* equivalent and was not converted\n", path, k)
	}
	return true, nil
}

func migrateConfig(cfgPath, graderPath string, out, errOut io.Writer) error {
	em := &envEmitter{values: map[string]string{}, errOut: errOut}

	var lc legacyConfig
	if found, err := decodeLegacy(cfgPath, &lc, errOut); err != nil {
		return err
	} else if found {
		em.set("KN_DATA_DIR", lc.Common.DataDir)
		if lc.Common.LogDir != "" {
			fmt.Fprintf(errOut, "notice: log_dir %q is not configurable anymore; logs go to %s/logs\n", lc.Common.LogDir, lc.Common.DataDir)
		}

		em.set("KN_DEBUG", lc.Common.Debug)
		em.set("KN_HOST_PREFIX", lc.Common.HostPrefix)
		em.set("KN_DB_DSN", lc.Common.DBDSN)
		if lc.Email.Enabled {
			em.set("KN_SMTP_HOST", lc.Email.Host)
			em.set("KN_SMTP_USERNAME", lc.Email.Username)
			em.set("KN_SMTP_PASSWORD", lc.Email.Password)
			em.set("KN_SMTP_FROM", lc.Email.SendAs)
		}
		em.set("KN_EVAL_MODE", lc.Eval.Mode)
		em.set("KN_EVAL_REMOTE_ENDPOINT", lc.Eval.Remote.Endpoint)
		em.set("KN_EVAL_REMOTE_TOKEN", lc.Eval.Remote.Token)
		em.set("KN_SANDBOX_NUM_CONCURRENT", lc.Eval.NumConcurrent)
		em.set("KN_SANDBOX_GLOBAL_MAX_MEM_KB", lc.Eval.GlobalMaxMem)
		em.set("KN_SANDBOX_STARTING_BOX", lc.Eval.StartingBox)

		// Admin-editable values go to the flags file, not the environment.
		if lc.Common.DefaultLang != "" {
			flags.DefaultLanguage.Update(lc.Common.DefaultLang)
			fmt.Fprintf(errOut, "flag %s = %q\n", flags.DefaultLanguage.InternalName(), lc.Common.DefaultLang)
		}
		if lc.Common.TestMaxMemKB != 0 {
			flags.TestMaxMemKB.Update(lc.Common.TestMaxMemKB)
			fmt.Fprintf(errOut, "flag %s = %d\n", flags.TestMaxMemKB.InternalName(), lc.Common.TestMaxMemKB)
		}
		if lc.Frontend.BannedHotProblems != nil {
			flags.BannedHotProblems.Update(lc.Frontend.BannedHotProblems)
			fmt.Fprintf(errOut, "flag %s = %v\n", flags.BannedHotProblems.InternalName(), lc.Frontend.BannedHotProblems)
		}
	}

	var lg legacyGrader
	if found, err := decodeLegacy(graderPath, &lg, errOut); err != nil {
		return err
	} else if found {
		g := lg.Grader
		em.set("KN_GRADER_LISTEN", g.Listen)
		em.set("KN_GRADER_TLS_CERT", g.CertFile)
		em.set("KN_GRADER_TLS_KEY", g.KeyFile)
		// The grader root is KN_DATA_DIR with scratch/ under it; derive the
		// root from the legacy scratch_dir unless config.toml already set one.
		if _, set := em.values["KN_DATA_DIR"]; !set && g.ScratchDir != "" {
			root := filepath.Dir(g.ScratchDir)
			em.set("KN_DATA_DIR", root)
			if filepath.Base(g.ScratchDir) != "scratch" {
				fmt.Fprintf(errOut, "notice: scratch_dir %q becomes %s/scratch; move or delete the old directory\n", g.ScratchDir, root)
			}
		}
		em.set("KN_GRADER_SCRATCH_TTL_SEC", g.ScratchTTLSec)
		em.set("KN_SANDBOX_NUM_CONCURRENT", g.NumConcurrent)
		em.set("KN_SANDBOX_GLOBAL_MAX_MEM_KB", g.GlobalMaxMem)
		em.set("KN_SANDBOX_STARTING_BOX", g.StartingBox)
		for _, cl := range g.Clients {
			if cl.Priority != "" {
				fmt.Fprintf(errOut, "warning: %s: client %q priority %q dropped (never consumed)\n", graderPath, cl.Name, cl.Priority)
			}
			em.set(config.GraderClientEnvPrefix+envName(cl.Name), cl.Token)
		}
	}

	em.write(out)
	return nil
}
