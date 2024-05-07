package box

import "github.com/KiloProjects/kilonova/internal/config"

var (
	EnsureCGKeeper    = config.GenFlag("feature.grader.ensure_keeper", false, "Ensure isolate-cg-keeper is running")
	IsolateConfigPath = config.GenFlag("feature.grader.isolate_config_path", "/usr/local/etc/isolate", "Configuration path for isolate sandbox")
)

func verifyKeeper() error {

	panic("TODO")
}

func startKeeper() error {
	panic("TODO")
}

func InitKeeper() error {
	if !EnsureCGKeeper.Value() {
		return nil
	}
	panic("TODO")
}
