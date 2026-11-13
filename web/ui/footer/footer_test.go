package footer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	. "maragu.dev/gomponents"
)

func TestView(t *testing.T) {
	ctx := context.WithValue(context.Background(), util.LangKey, "ro")

	t.Run("selects the session language", func(t *testing.T) {
		got := render(t, View(ctx, 0, time.Now()))
		if !strings.Contains(got, `<option value="ro" selected>`) || strings.Contains(got, `<option value="en" selected>`) {
			t.Fatalf("wrong language selected: %s", got)
		}
	})

	t.Run("donate link follows its flag", func(t *testing.T) {
		defer flags.DonationsEnabled.Update(t.Context(), flags.DonationsEnabled.Value())
		flags.DonationsEnabled.Update(t.Context(), false)
		if got := render(t, View(ctx, 0, time.Now())); strings.Contains(got, `href="/donate"`) {
			t.Fatalf("donate link shown while disabled")
		}
	})

	t.Run("query count is admin-only", func(t *testing.T) {
		if got := render(t, View(ctx, 0, time.Now())); !strings.Contains(got, "<details") || strings.Contains(got, "~") {
			t.Fatalf("timings missing or query count shown to visitor: %s", got)
		}
	})
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}
