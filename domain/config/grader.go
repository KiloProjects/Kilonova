package config

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
)

// GraderConf is the remote grader process's own configuration (kn grader-serve).
// Its root directory is Common.DataDir, holding scratch/ (served over the HTTP
// /scratch endpoint) and logs/. Sandbox capacity comes from the shared Eval
// fields; clients come from GraderClientsFromEnv.
type GraderConf struct {
	Listen   string // host:port for the grader HTTP server
	CertFile string // TLS server cert
	KeyFile  string // TLS server key

	// ScratchTTLSec is the orphan GC TTL; must be >> max eval duration.
	ScratchTTLSec int
}

// GraderClientConf is one entry in the token registry: a named platform client
// with its bearer token.
type GraderClientConf struct {
	Name  string
	Token string
}

func (conf GraderClientConf) Compare(other GraderClientConf) int {
	return cmp.Compare(conf.Name, other.Name)
}

// GraderClientEnvPrefix is the per-client variable prefix: KN_GRADER_CLIENT_<NAME>=<token>.
const GraderClientEnvPrefix = "KN_GRADER_CLIENT_"

// GraderClientsFromEnv builds the client registry from every
// KN_GRADER_CLIENT_<NAME>=<token> variable in environ (as returned by
// os.Environ). NAME is lowercased to form the client name. An empty registry
// is an error: a grader nobody can talk to is a misconfiguration.
func GraderClientsFromEnv(environ []string) ([]GraderClientConf, error) {
	var out []GraderClientConf
	for _, kv := range environ {
		key, token, _ := strings.Cut(kv, "=")
		name, ok := strings.CutPrefix(key, GraderClientEnvPrefix)
		if !ok {
			continue
		}
		if name == "" || token == "" {
			return nil, fmt.Errorf("%s: expected %s<NAME>=<token> with a non-empty name and token", key, GraderClientEnvPrefix)
		}
		out = append(out, GraderClientConf{Name: strings.ToLower(name), Token: token})
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no %s<NAME>=<token> variables set; the grader needs at least one client", GraderClientEnvPrefix)
	}

	slices.SortFunc(out, GraderClientConf.Compare)
	return out, nil
}
