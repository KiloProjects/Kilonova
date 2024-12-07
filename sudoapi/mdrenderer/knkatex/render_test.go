package knkatex_test

import (
	"bytes"
	"context"
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
			knkatex.Render(context.Background(), &buf, []byte(test.Source), false)
			knkatex.Render(context.Background(), &buf, []byte(test.Source), true)
		}
	}
}
