package ui

import (
	"context"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/internal/util"
)

func TestT(t *testing.T) {
	en := render(t, T(context.WithValue(context.Background(), util.LangKey, "en"), "contests"))
	ro := render(t, T(context.WithValue(context.Background(), util.LangKey, "ro"), "contests"))
	if en == "" || en == "contests" || en == ro {
		t.Fatalf("not translated per language: en=%q ro=%q", en, ro)
	}
}

func TestTRaw(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	if raw, esc := render(t, TRaw(ctx, "unverifiedEmail")), render(t, T(ctx, "unverifiedEmail")); !strings.Contains(raw, "<a") || strings.Contains(esc, "<a") {
		t.Fatalf("TRaw should keep markup, T should escape it: raw=%q esc=%q", raw, esc)
	}
}
