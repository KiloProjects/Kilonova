package adminviews

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
)

var enCtx = context.WithValue(context.Background(), util.LangKey, "en")

func TestPanel(t *testing.T) {
	t.Run("renders flags with original values and the authed user id in the script", func(t *testing.T) {
		got := render(t, Panel(enCtx, PanelParams{
			AuthedUserID:  7,
			BoolFlags:     []config.Flag[bool]{config.GenFlag("test.adminviews.bool", true, "Bool <b>")},
			IntArrayFlags: []config.Flag[[]int]{config.GenFlag("test.adminviews.ints", []int{1, 2}, "Ints")},
		}))
		must(t, got, `data-name="test.adminviews.bool"`, `data-org_value="true"`, "checked", "Bool &lt;b&gt;",
			`data-org_value="1,2"`, `value="1,2"`, `user.id != 1 && 7 == 1`, `id="stringFlagsDiv"`, "N/A", "`/user/byName/${user}`")
	})
}

func TestUserGen(t *testing.T) {
	t.Run("selects the default language", func(t *testing.T) {
		got := render(t, UserGen(enCtx, "en"))
		must(t, got, `<option value="en" selected>`, `<option value="ro">`)
	})
}

func TestAuditLog(t *testing.T) {
	t.Run("author kinds", func(t *testing.T) {
		got := render(t, AuditLog(enCtx, AuditLogParams{Logs: []*kilonova.AuditLog{
			{ID: 1, Message: "<x>", Author: &kilonova.UserBrief{ID: 3, Name: "bob"}},
			{ID: 2, SystemLog: true},
			{ID: 3},
		}}))
		must(t, got, `<a href="/profile/bob">bob</a> (#3)`, "fa-cog", "fa-question-square", "&lt;x&gt;")
	})
}

func TestDebug(t *testing.T) {
	t.Run("bucket actions and metric values", func(t *testing.T) {
		f, n := 1.5, uint64(42)
		got := render(t, Debug(enCtx, DebugParams{
			Metrics: []*Metric{{Name: "a", Float: &f}, {Name: "b", Int: &n}, {Name: "c"}},
			BucketStats: []*datastore.BucketStats{
				{Name: `ev"il`, MaxTTL: time.Hour, Cache: true},
				{Name: "persist", Persistent: true, MaxSize: 4096},
			},
		}))
		must(t, got, ">1.5<", ">42<", "???", `onclick="evictBucket(&#34;ev\&#34;il&#34;)"`, "cleanBucket", "1h0m0s", "Persist")
		if c := strings.Count(got, "evictBucket("); c != 2 { // one in the script, one for the non-persistent bucket
			t.Fatalf("want evict button only on the non-persistent bucket, got %d occurrences", c)
		}
	})
}

func TestGraderInfo(t *testing.T) {
	got := render(t, GraderInfo(enCtx, []*sudoapi.GraderLanguage{{Name: "C++", Command: "g++ <x>", Version: "13"}}))
	must(t, got, "<code>g++ &lt;x&gt;</code>", "<code>13</code>")
}

func TestContestGen(t *testing.T) {
	t.Run("posts a multipart bundle with a dry run toggle", func(t *testing.T) {
		got := render(t, ContestGen(enCtx))
		must(t, got, `action="/admin/contest_gen"`, `enctype="multipart/form-data"`, `name="bundle"`, `name="dryRun"`)
	})
}
