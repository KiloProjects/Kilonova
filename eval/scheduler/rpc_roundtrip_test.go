package scheduler

import (
	"context"
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

// TestScratchRoundTripOverConversion drives SaveFile -> RunBox3 -> ReadFile ->
// DeleteFile through the exact proto conversion helpers used on the wire, so a
// dropped/renamed field fails here instead of silently in production.
func TestScratchRoundTripOverConversion(t *testing.T) {
	sc := scratch.New(afero.NewMemMapFs())
	const payload = "hello grader"

	inID, err := sc.SaveFile(strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}

	req := &eval.Box3Request{
		InputFiles: []eval.ScratchFile{{Identifier: inID, BoxPath: "/box/in.txt", Mode: 0o644}},
		Command:    []string{"/bin/cat", "/box/in.txt"},
		RunConfig:  &eval.RunConfig{MemoryLimit: 65536, TimeLimit: 1.5, EnvToSet: map[string]string{"A": "b"}},
	}

	// Client side: eval -> proto. Server side: proto -> eval, run, eval -> proto.
	wireReq := box3RequestToProto(req)
	server := &GraderServer{sched: &echoSched{scratch: sc}}
	resp, err := server.sched.RunBox3(context.Background(), box3RequestFromProto(wireReq), 0)
	if err != nil {
		t.Fatal(err)
	}
	wireResp := box3ResponseToProto(resp)
	// Client side: proto -> eval.
	got := box3ResponseFromProto(wireResp)

	outID, ok := got.Files["/box/in.txt"]
	if !ok {
		t.Fatalf("output identifier missing from response files: %v", got.Files)
	}
	if got.Stats.Status != "OK" || got.Stats.Time != 0.5 {
		t.Fatalf("stats did not survive conversion: %+v", got.Stats)
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

// TestRunConfigConversionIsLossless guards the widest message against field drift.
func TestRunConfigConversionIsLossless(t *testing.T) {
	in := &eval.RunConfig{
		StderrToStdout: true,
		InputPath:      "/box/in", OutputPath: "/box/out", StderrPath: "/box/err",
		MemoryLimit: 131072, TimeLimit: 2.0, WallTimeLimit: 5.0,
		InheritEnv: true, EnvToInherit: []string{"PATH"}, EnvToSet: map[string]string{"K": "V"},
		EnableInternet: true,
		Directories:    []language.Directory{{In: "/a", Out: "/b", Opts: "rw", Removes: true, Verbatim: true}},
	}
	got := runConfigFromProto(runConfigToProto(in))
	if !reflect.DeepEqual(in, got) {
		t.Fatalf("RunConfig changed across conversion:\n in: %+v\ngot: %+v", in, got)
	}
}
