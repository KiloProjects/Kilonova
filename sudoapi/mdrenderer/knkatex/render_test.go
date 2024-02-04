package knkatex_test

import (
	"bytes"
	"strings"
	"testing"
	"unicode/utf8"

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

func BenchmarkRenderGoja(b *testing.B) {
	tests := getCases()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, test := range tests {
			var buf bytes.Buffer
			knkatex.RenderGoja(&buf, []byte(test.Source), false)
			knkatex.RenderGoja(&buf, []byte(test.Source), true)
		}
	}
}

func BenchmarkQuickJS(b *testing.B) {
	tests := getCases()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, test := range tests {
			var buf bytes.Buffer
			knkatex.RenderQuickJS(&buf, []byte(test.Source), false)
			knkatex.RenderQuickJS(&buf, []byte(test.Source), true)
		}
	}
}

func TestRenderers(t *testing.T) {
	for _, c := range getCases() {
		c := c
		t.Run(c.Name+"-display=false", func(t *testing.T) {
			t.Parallel()
			var buf1, buf2 bytes.Buffer
			err1 := knkatex.RenderGoja(&buf1, []byte(c.Source), false)
			err2 := knkatex.RenderQuickJS(&buf2, []byte(c.Source), false)
			if (err1 == nil && err2 != nil) || (err1 != nil && err2 == nil) {
				t.Fatalf("Expected same error value err1: %#v err2: %#v", err1, err2)
			}
			if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
				t.Fatalf("Expected identical strings but got different results")
			}
		})

		t.Run(c.Name+"-display=true", func(t *testing.T) {
			t.Parallel()
			var buf1, buf2 bytes.Buffer
			err1 := knkatex.RenderGoja(&buf1, []byte(c.Source), true)
			err2 := knkatex.RenderQuickJS(&buf2, []byte(c.Source), true)
			if (err1 == nil && err2 != nil) || (err1 != nil && err2 == nil) {
				t.Fatalf("Expected same error value err1: %#v err2: %#v", err1, err2)
			}
			if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
				t.Fatalf("Expected identical strings but got different results")
			}
		})
	}
}

func FuzzRenderer(f *testing.F) {
	for _, c := range getCases() {
		f.Add(c.Source)
	}
	f.Fuzz(func(t *testing.T, src string) {
		src = strings.ReplaceAll(src, "\x00", "")
		if !utf8.ValidString(src) {
			t.Skip("Invalid fuzz")
		}
		var buf1, buf2 bytes.Buffer
		err1 := knkatex.RenderGoja(&buf1, []byte(src), false)
		err2 := knkatex.RenderQuickJS(&buf2, []byte(src), false)
		if !bytes.Equal(buf1.Bytes(), buf2.Bytes()) {
			t.Fatalf("%q: Expected identical strings but got different results. buf1: %q, buf2: %q", src, buf1.String(), buf2.String())
		}
		if (err1 == nil && err2 != nil) || (err1 != nil && err2 == nil) {
			t.Fatalf("%q: Expected same error value err1: %#v err2: %#v", src, err1, err2)
		}
	})
}
