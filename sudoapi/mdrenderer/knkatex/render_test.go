package knkatex_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer/knkatex"
)

type testCase struct {
	Name   string
	Source string
}

func getCases() []testCase {
	return []testCase{
		{Name: "simple text", Source: `\text{Hello world!}`},
		{Name: "assignment", Source: `y_1=y_2`},
		{Name: "complex equation", Source: `f(\relax{x}) = \int_{-\infty}^\infty \hat{f}(\xi)\,e^{2 \pi i \xi x} \,d\xi`},
		{Name: "restriction", Source: `1 \leq x_1, y_1, x_2, y_2 \leq 1 \ 000 \ 000 \ 000`},
		{Name: "error", Source: `x\text{`},
	}
}

func BenchmarkRender(b *testing.B) {
	tests := getCases()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, test := range tests {
			var buf bytes.Buffer
			knkatex.Render(b.Context(), &buf, []byte(test.Source), false)
			knkatex.Render(b.Context(), &buf, []byte(test.Source), true)
		}
	}
}

func TestRender(t *testing.T) {
	for _, test := range getCases() {
		t.Run(test.Name, func(t *testing.T) {
			for _, display := range []bool{false, true} {
				var buf bytes.Buffer
				err := knkatex.Render(t.Context(), &buf, []byte(test.Source), display)
				out := buf.String()
				if test.Name == "error" {
					// Malformed input is reported inline rather than rendered.
					if !strings.Contains(out, "<code>") {
						t.Errorf("display=%v: expected the error to be reported, got %q", display, out)
					}
					continue
				}
				if err != nil {
					t.Errorf("display=%v: %v", display, err)
				}
				if !strings.Contains(out, "class=\"katex") {
					t.Errorf("display=%v: no KaTeX markup: %.120q", display, out)
				}
			}
		})
	}
}
