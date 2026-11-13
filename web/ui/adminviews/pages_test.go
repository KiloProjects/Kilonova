package adminviews

import (
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/domain/datastore"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

var enCtx = uitest.Ctx("en")

func TestPanel(t *testing.T) {
	t.Run("renders flags with original values and the authed user id in the script", func(t *testing.T) {
		got := uitest.Render(t, Panel(enCtx, PanelParams{
			AuthedUserID:  7,
			BoolFlags:     []config.Flag[bool]{config.GenFlag("test.adminviews.bool", true, "Bool <b>")},
			IntArrayFlags: []config.Flag[[]int]{config.GenFlag("test.adminviews.ints", []int{1, 2}, "Ints")},
			StrArrayFlags: []config.Flag[[]string]{config.GenFlag("test.adminviews.strs", []string{`a,"b`}, "Strs")},
		}))
		uitest.Contains(t, got, `data-name="test.adminviews.bool"`, `data-org_value="true"`, "checked", "Bool &lt;b&gt;",
			`data-org_value="[1,2]"`, `value="[1,2]"`, `value="[&#34;a,\&#34;b&#34;]"`, `user.id != 1 && 7 == 1`, `id="stringFlagsDiv"`, "N/A", "`/user/byName/${user}`")
	})
}

func TestUserGen(t *testing.T) {
	t.Run("selects the default language", func(t *testing.T) {
		got := uitest.Render(t, UserGen(enCtx, "en"))
		uitest.Contains(t, got, `<option value="en" selected>`, `<option value="ro">`)
	})
}

func TestAuditLog(t *testing.T) {
	t.Run("author kinds", func(t *testing.T) {
		got := uitest.Render(t, AuditLog(enCtx, AuditLogParams{Logs: []*kilonova.AuditLog{
			{ID: 1, Message: "<x>", Author: &kilonova.UserBrief{ID: 3, Name: "bob"}},
			{ID: 2, SystemLog: true},
			{ID: 3},
		}}))
		uitest.Contains(t, got, `<a href="/profile/bob">bob</a> (#3)`, "fa-cog", "fa-question-square", "&lt;x&gt;")
	})
}

func TestDebug(t *testing.T) {
	t.Run("bucket actions and metric values", func(t *testing.T) {
		f, n := 1.5, uint64(42)
		got := uitest.Render(t, Debug(enCtx, DebugParams{
			Metrics: []*Metric{{Name: "a", Float: &f}, {Name: "b", Int: &n}, {Name: "c"}},
			BucketStats: []*datastore.BucketStats{
				{Name: `ev"il`, MaxTTL: time.Hour, Cache: true},
				{Name: "persist", Persistent: true, MaxSize: 4096},
			},
		}))
		uitest.Contains(t, got, ">1.5<", ">42<", "???", `onclick="evictBucket(&#34;ev\&#34;il&#34;)"`, "cleanBucket", "1h0m0s", "Persist")
		// one in the script, one for the non-persistent bucket
		require.Equal(t, 2, strings.Count(got, "evictBucket("), "want evict button only on the non-persistent bucket")
	})
}

func TestGraderInfo(t *testing.T) {
	got := uitest.Render(t, GraderInfo(enCtx, []*sudoapi.GraderLanguage{{Name: "C++", Command: "g++ <x>", Version: "13"}}))
	uitest.Contains(t, got, "<code>g++ &lt;x&gt;</code>", "<code>13</code>")
}

func TestContestGen(t *testing.T) {
	t.Run("posts a multipart bundle with a dry run toggle", func(t *testing.T) {
		got := uitest.Render(t, ContestGen(enCtx))
		uitest.Contains(t, got, `action="/admin/contest_gen"`, `enctype="multipart/form-data"`, `name="bundle"`, `name="dryRun"`)
	})
}

func TestTabs(t *testing.T) {
	got := uitest.Render(t, Tabs(enCtx, "/admin/user_gen"))
	uitest.Contains(t, got, `<a class="p-1 topbar-selected" href="/admin/user_gen">`, `href="/admin/contest_gen"`, `href="/admin/sessions"`, `<a class="p-1" href="/admin">`)
}
