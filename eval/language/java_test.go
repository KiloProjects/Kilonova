package language

import (
	"strings"
	"testing"
)

func TestJavaFilenameIsNeverShellCode(t *testing.T) {
	j := Java{}
	for in, want := range map[string]string{
		"":                 "/box/Main.java",
		"Sol.java":         "/box/Sol.java",
		"../../etc/x.java": "/box/x.java",
		"..":               "/box/Main.java",
		"noext":            "/box/Main.java",
	} {
		if got := j.SourceName(in); got != want {
			t.Errorf("SourceName(%q) = %q, want %q", in, got, want)
		}
	}

	evil := j.SourceName("$(id)`id`;x.java")
	cmd := j.CompileCommand([]string{evil})
	if cmd[0] != "sh" || cmd[1] != "-c" {
		t.Fatalf("unexpected command shape: %q", cmd)
	}
	if strings.Contains(cmd[2], "id") {
		t.Errorf("file name leaked into the script text: %q", cmd[2])
	}
	if cmd[len(cmd)-1] != evil {
		t.Errorf("file name should be passed through as a plain argument, got %q", cmd)
	}
}
