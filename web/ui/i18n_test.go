package ui

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

func TestT(t *testing.T) {
	en := uitest.Render(t, T(context.WithValue(context.Background(), util.LangKey, "en"), "contests"))
	ro := uitest.Render(t, T(context.WithValue(context.Background(), util.LangKey, "ro"), "contests"))
	require.NotEmpty(t, en)
	require.NotEqual(t, "contests", en)
	require.NotEqual(t, ro, en, "not translated per language")
}

func TestTRaw(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "en")
	require.Contains(t, uitest.Render(t, TRaw(ctx, "unverifiedEmail")), "<a", "TRaw should keep markup")
	require.NotContains(t, uitest.Render(t, T(ctx, "unverifiedEmail")), "<a", "T should escape markup")
}
