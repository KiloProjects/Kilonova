package adminviews

import (
	"slices"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova/domain/config"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
	g "maragu.dev/gomponents"
)

// waitingForGroup lists the flags not yet placed in a grouped admin section (settingsGroups)
// or a dedicated page. It may only shrink: TestEveryFlagHasAHome fails when a flag is on it
// and also grouped, and when a new flag is on neither.
var waitingForGroup = []string{
	"admin.important_webhook",
	"admin.mailer.branding",
	"admin.verbose_webhook",
	"behavior.contests.anyone_virtual",
	"behavior.contests.normal_user_max_day",
	"behavior.everyone_subs",
	"behavior.everyone_subs.blacklist",
	"behavior.force_authed",
	"behavior.problem.default_source_size",
	"behavior.problems.lockdown_published",
	"behavior.sessions.max_concurrent",
	"behavior.submissions.max_viewing_count",
	"behavior.submissions.source_size_whitelist",
	"behavior.submissions.user_max_minute",
	"behavior.submissions.user_max_unverified",
	"behavior.submissions.user_max_waiting",
	"behavior.user_agent_filter",
	"eval.test_max_mem_kb",
	"feature.analytics.goat_prefix",
	"feature.captcha.enabled",
	"feature.captcha.min_trigger",
	"feature.external_resources.enabled",
	"feature.frontend.all_subs_page",
	"feature.frontend.anyone_tags",
	"feature.frontend.ascending_contests",
	"feature.frontend.feedback_url",
	"feature.frontend.sentry",
	"feature.frontend.sentry_dsn",
	"feature.frontend.view_other_profiles",
	"feature.pastes.enabled",
	"feature.platform.signup",
	"feature.username_changes.enabled",
	"frontend.default_language",
	"frontend.donation.bmac_webhook_secret",
	"frontend.problem_statistics",
	"integrations.discord.publish_announcement_channel",
	"integrations.moss.user_id",
	"oidc.cryptoKey",
	"oidc.rsaKey.id",
	"oidc.rsaKey.private",
}

// element is a parsed element with the chain of its ancestors, innermost first.
type element struct {
	node      *html.Node
	ancestors []*html.Node
}

func (e element) attr(name string) (string, bool) { return attr(e.node, name) }

func attr(n *html.Node, name string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == name {
			return a.Val, true
		}
	}
	return "", false
}

// insideDetails reports whether e sits in a <details> with the given id.
func (e element) insideDetails(id string) bool {
	return slices.ContainsFunc(e.ancestors, func(a *html.Node) bool {
		v, _ := attr(a, "id")
		return a.Data == "details" && v == id
	})
}

// parse renders n and returns every element in document order.
func parse(t *testing.T, n g.Node) []element {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(uitest.Render(t, n)))
	require.NoError(t, err)
	var out []element
	var walk func(n *html.Node, ancestors []*html.Node)
	walk = func(n *html.Node, ancestors []*html.Node) {
		if n.Type == html.ElementNode {
			out = append(out, element{n, ancestors})
			ancestors = append([]*html.Node{n}, ancestors...)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, ancestors)
		}
	}
	walk(doc, nil)
	return out
}

// flagKind is the data-type the inputs of the flag named key must carry.
func flagKind(key string) string {
	switch {
	case has[bool](key):
		return kindBool
	case has[string](key):
		return kindString
	case has[int](key):
		return kindInt
	case has[[]int](key):
		return kindIntList
	case has[[]string](key):
		return kindStrList
	}
	return ""
}

func has[T any](key string) bool { _, ok := config.GetFlag[T](key); return ok }

func TestSettings(t *testing.T) {
	t.Run("one typed input per grouped flag", func(t *testing.T) {
		inputs := map[string]string{}
		for _, el := range parse(t, Settings(enCtx)) {
			if key, ok := el.attr("data-flag"); ok && el.node.Data == "input" {
				kind, _ := el.attr("data-type")
				require.NotContains(t, inputs, key, "flag %s rendered twice", key)
				inputs[key] = kind
			}
		}
		for _, grp := range settingsGroups {
			for _, key := range grp.flags {
				require.Contains(t, inputs, key)
				require.Equal(t, flagKind(key), inputs[key], key)
			}
		}
	})
}

func TestFlatFlags(t *testing.T) {
	params := PanelParams{
		BoolFlags:     config.GetFlags[bool](),
		StringFlags:   config.GetFlags[string](),
		IntFlags:      config.GetFlags[int](),
		IntArrayFlags: config.GetFlags[[]int](),
		StrArrayFlags: config.GetFlags[[]string](),
	}
	elems := parse(t, Panel(enCtx, params))

	var secretsSeen, total int
	for _, el := range elems {
		if el.node.Data == "details" {
			if id, _ := el.attr("id"); id == "flatFlags" || id == "secretFlags" {
				_, open := el.attr("open")
				require.False(t, open, "%s must start collapsed", id)
			}
		}
		key, ok := el.attr("data-flag")
		if !ok || el.node.Data != "input" {
			continue
		}
		total++
		kind, _ := el.attr("data-type")
		require.Equal(t, flagKind(key), kind, "%s needs data-type for flags.js", key)
		require.True(t, el.insideDetails("flatFlags"), "%s must be in the collapsed fallback", key)

		f, _ := config.GetFlag[string](key) // every secret flag today is a string flag
		secret := f != nil && f.Secret()
		require.Equal(t, secret, el.insideDetails("secretFlags"), "%s: secret flags, and only they, sit in the nested section", key)
		if secret {
			secretsSeen++
		}
	}
	require.Equal(t, len(config.FlagNames()), total, "the fallback lists every flag")
	require.Positive(t, secretsSeen)
}

// TestEveryFlagHasAHome fails when a registered flag is neither on a grouped admin page
// nor on waitingForGroup, or is on both, or is rendered twice. Flags registered by tests
// ("test." prefix) are ignored.
func TestEveryFlagHasAHome(t *testing.T) {
	placed := map[string]int{}
	for _, page := range []g.Node{Settings(enCtx), Flashes(enCtx)} {
		for _, el := range parse(t, page) {
			if key, ok := el.attr("data-flag"); ok {
				placed[key]++
			}
		}
	}
	for key, n := range placed {
		require.Equal(t, 1, n, "flag %s has more than one place in the grouped admin UI", key)
		require.NotContains(t, waitingForGroup, key, "flag %s is grouped now: remove it from waitingForGroup", key)
	}
	for _, key := range config.FlagNames() {
		if strings.HasPrefix(key, "test.") {
			continue
		}
		if placed[key] == 0 {
			require.Contains(t, waitingForGroup, key, "flag %s has no place in the admin UI: add it to a settings group or to waitingForGroup", key)
		}
	}
	for _, key := range waitingForGroup {
		require.Contains(t, config.FlagNames(), key, "waitingForGroup lists %s, which is no longer a flag", key)
	}
}
