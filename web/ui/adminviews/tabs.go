package adminviews

import (
	"context"

	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Tabs is the admin section bar shown above every /admin page; path selects the current tab.
func Tabs(ctx context.Context, path string) Node {
	tab := func(href, key string) ui.Tab { return ui.Tab{Key: href, Href: href, Label: ui.T(ctx, key)} }
	return Nav(Class("segment-panel"), ui.Tabs(path,
		tab("/admin", "panel.admin"),
		tab("/admin/users", "users"),
		tab("/admin/sessions", "manage_sessions"),
		tab("/admin/oauth", "oauth_clients"),
		tab("/admin/auditLog", "panel.audit_log"),
		tab("/admin/debug", "panel.debug"),
		tab("/admin/user_gen", "user_gen.title"),
		tab("/admin/contest_gen", "contest_gen.title"),
		tab("/grader", "panel.grader"),
	))
}
