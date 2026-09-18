package scheduler

import (
	"context"
	"encoding/json"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/KiloProjects/kilonova/eval/language"
	"github.com/KiloProjects/kilonova/eval/scratch"
	"github.com/spf13/afero"
)

// echoSched is a stub Box3Scheduler: it copies each input scratch file to a new
// output identifier, mimicking a box that produces outputs from inputs.
type echoSched struct{ scratch eval.Scratch }

func (e *echoSched) RunBox3(ctx context.Context, req *eval.Box3Request, memQuota int64) (*eval.Box3Response, error) {
	files := make(map[string]string)
	for _, in := range req.InputFiles {
		rc, err := e.scratch.ReadFile(in.Identifier)
		if err != nil {
			return nil, err
		}
		id, err := e.scratch.SaveFile(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}
		files[in.BoxPath] = id
	}
	return &eval.Box3Response{Stats: &eval.RunStats{Status: "OK", Time: 0.5}, Files: files}, nil
}

func (e *echoSched) RunMultibox3(context.Context, *eval.Multibox3Request, int64, int64) (*eval.Box3Response, []*eval.RunStats, error) {
	return nil, nil, nil
}
func (e *echoSched) Close(context.Context) error { return nil }

// TestScratchRoundTripOverWire drives SaveFile -> RunBox3 -> ReadFile ->
// DeleteFile through the real GraderClient/GraderServer pair over httptest, so a
// field that doesn't survive JSON fails here instead of silently in production.
func TestScratchRoundTripOverWire(t *testing.T) {
	sc := scratch.New(afero.NewMemMapFs())
	const payload = "hello grader"

	inID, err := sc.SaveFile(strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}

	url := startTestGrader(t, &echoSched{scratch: sc}, nil, "tok")
	got, err := NewGraderClient(nil, url, "tok").RunBox3(context.Background(), &eval.Box3Request{
		InputFiles: []eval.ScratchFile{{Identifier: inID, BoxPath: "/box/in.txt", Mode: 0o644}},
		Command:    []string{"/bin/cat", "/box/in.txt"},
		RunConfig:  &eval.RunConfig{MemoryLimit: 65536, TimeLimit: 1.5, EnvToSet: map[string]string{"A": "b"}},
	}, 0)
	if err != nil {
		t.Fatal(err)
	}

	outID, ok := got.Files["/box/in.txt"]
	if !ok {
		t.Fatalf("output identifier missing from response files: %v", got.Files)
	}
	if got.Stats.Status != "OK" || got.Stats.Time != 0.5 {
		t.Fatalf("stats did not survive the wire: %+v", got.Stats)
	}

	rc, err := sc.ReadFile(outID)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := io.ReadAll(rc)
	rc.Close()
	if string(data) != payload {
		t.Fatalf("round-tripped payload = %q, want %q", data, payload)
	}

	if err := sc.DeleteFile(outID); err != nil {
		t.Fatal(err)
	}
	if _, err := sc.ReadFile(outID); err == nil {
		t.Fatal("identifier still readable after DeleteFile")
	}
}

// TestRunConfigJSONIsLossless guards the widest message against JSON surprises.
func TestRunConfigJSONIsLossless(t *testing.T) {
	in := &eval.Box3Request{
		InputFiles: []eval.ScratchFile{{Identifier: "id", BoxPath: "/box/in", Mode: 0o600}},
		Command:    []string{"/bin/true", "-x"},
		RunConfig: &eval.RunConfig{
			StderrToStdout: true,
			InputPath:      "/box/in", OutputPath: "/box/out", StderrPath: "/box/err",
			MemoryLimit: 131072, TimeLimit: 2.0, WallTimeLimit: 5.0,
			InheritEnv: true, EnvToInherit: []string{"PATH"}, EnvToSet: map[string]string{"K": "V"},
			EnableInternet: true,
			Directories:    []language.Directory{{In: "/a", Out: "/b", Opts: "rw", Removes: true, Verbatim: true}},
		},
		OutputFilePaths: []string{"/box/out"},
	}
	buf, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	var got eval.Box3Request
	if err := json.Unmarshal(buf, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, &got) {
		t.Fatalf("Box3Request changed across JSON:\n in: %+v\ngot: %+v", in, &got)
	}
}
