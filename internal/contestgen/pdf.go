package contestgen

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed printable_contestants.typ
var printableTemplate []byte

// PrintablePDF renders a teamsOut.csv (as produced by Result.Files) into a printable PDF of credentials using Typst.
func PrintablePDF(ctx context.Context, teamsCSV []byte) ([]byte, error) {
	if _, err := exec.LookPath("typst"); err != nil {
		return nil, errors.New("typst compiler not found")
	}

	dir, err := os.MkdirTemp("", "kn-printable-contestants-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "in.csv"), teamsCSV, 0600); err != nil {
		return nil, fmt.Errorf("write temporary CSV: %w", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "templ.typ"), printableTemplate, 0600); err != nil {
		return nil, fmt.Errorf("write temporary template: %w", err)
	}

	var stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, "typst", "compile", "templ.typ", "out.pdf")
	cmd.Dir = dir
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("typst compile failed: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return os.ReadFile(filepath.Join(dir, "out.pdf"))
}
