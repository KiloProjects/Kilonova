package submissions

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func TestSubmissionTestLinks(t *testing.T) {
	testID := 55
	subtests := []*kilonova.SubTest{
		{ID: 11, VisibleID: 1, TestID: &testID, Done: true},
		{ID: 12, VisibleID: 2, TestID: &testID, Done: true},
		{ID: 13, VisibleID: 3, TestID: &testID},
		{ID: 14, VisibleID: 4, TestID: nil, Done: true, Skipped: true},
	}
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	params := SubmissionTestTableParams{Subtests: subtests, SubType: kilonova.EvalTypeClassic}
	var out bytes.Buffer
	if err := SubmissionTestTable(params).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"/assets/subtest/", "/assets/problem/", "/edit/test/"} {
		if strings.Contains(out.String(), forbidden) {
			t.Fatalf("test link %q shown without permission: %s", forbidden, out.String())
		}
	}

	access := &SubmissionTestAccess{ProblemID: 7, OutputAvailable: map[int]bool{11: true, 13: true, 14: true}}
	params.TestAccess = access
	out.Reset()
	if err := SubmissionTestTable(params).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "/edit/test/") {
		t.Fatalf("test edit link shown to non-editor: %s", out.String())
	}
	for _, want := range []string{"/assets/problem/7/test/55/input", "/assets/problem/7/test/55/output", "/assets/subtest/11"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing viewer link %q in %s", want, out.String())
		}
	}

	access.CanEdit = true
	out.Reset()
	if err := SubmissionTestTable(params).Render(ctx, &out); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/problems/7/edit/test/55", "/assets/problem/7/test/55/input", "/assets/problem/7/test/55/output", "/assets/subtest/11", "out (unavailable)", "out (pending)", "retained only temporarily"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("missing %q in %s", want, out.String())
		}
	}
	for _, forbidden := range []string{"/assets/subtest/12", "/assets/subtest/13", "/assets/subtest/14", "/problems/7/edit/test/0"} {
		if strings.Contains(out.String(), forbidden) {
			t.Errorf("unexpected %q in %s", forbidden, out.String())
		}
	}
}
