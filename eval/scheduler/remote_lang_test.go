package scheduler_test

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/scheduler"
	"github.com/KiloProjects/kilonova/eval/scratch"
)

// TestRemoteLanguages runs a+b in a real sandbox on a live `kn grader-serve`, through
// the same client/scratch/Box2Wrapper path the platform uses in remote mode.
// Skipped unless KN_TEST_GRADER_ENDPOINT and KN_TEST_GRADER_TOKEN are set.
func TestRemoteLanguages(t *testing.T) {
	endpoint, token := os.Getenv("KN_TEST_GRADER_ENDPOINT"), os.Getenv("KN_TEST_GRADER_TOKEN")
	if endpoint == "" || token == "" {
		t.Skip("KN_TEST_GRADER_ENDPOINT / KN_TEST_GRADER_TOKEN not set")
	}
	ctx := context.Background()
	httpClient := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}} //nolint:gosec // test grader is self-signed

	client := scheduler.NewGraderClient(httpClient, endpoint, token)
	langMgr, err := scheduler.NewRemoteLanguageManager(ctx, client, slog.Default())
	if err != nil {
		t.Fatal(err)
	}
	sc, err := scratch.NewHTTP(httpClient, endpoint+"/scratch", token)
	if err != nil {
		t.Fatal(err)
	}
	runner := scheduler.NewBox2Wrapper(sc, nil, client)

	for name, code := range map[string]string{
		"julia": "a, b = parse.(Int, split(readline()))\nprintln(a + b)\n",
	} {
		t.Run(name, func(t *testing.T) {
			lang := langMgr.Language(name)
			if lang == nil {
				t.Fatalf("grader does not report %s", name)
			}
			t.Logf("version: %s", langMgr.LanguageVersions(ctx)[name])

			memLimit := language.MemoryLimit(lang, 64*1024) // a typical 64MB problem
			resp, err := runner.RunBox2(ctx, &eval.Box2Request{
				InputByteFiles: map[string]*eval.ByteFile{
					lang.SourceName(""): {Data: []byte(code), Mode: 0o666},
					"/box/stdin":        {Data: []byte("2 3\n"), Mode: 0o666},
				},
				Command: lang.RunCommand([]string{lang.ExecuteName("")}, memLimit),
				RunConfig: &eval.RunConfig{
					EnvToSet:       lang.RunEnv(),
					Directories:    lang.Mounts(),
					MemoryLimit:    memLimit,
					TimeLimit:      1 * max(lang.TimeLimitMultiplier(), 1),
					WallTimeLimit:  10,
					InputPath:      "/box/stdin",
					OutputPath:     "/box/stdout",
					StderrToStdout: true,
				},
				OutputByteFiles: []string{"/box/stdout"},
			}, 0)
			if err != nil {
				t.Fatal(err)
			}
			out := string(resp.ByteFiles["/box/stdout"])
			t.Logf("stats: %+v", *resp.Stats)
			if resp.Stats.Status != "" && resp.Stats.Status != "OK" || strings.TrimSpace(out) != "5" {
				t.Fatalf("got status %q, output %q", resp.Stats.Status, out)
			}
		})
	}
}
