// Package navbar renders the site-wide top navigation bar.
package navbar

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

const mobileButtonClass = "text-gray-500 dark:text-gray-300 hover:text-black focus:text-black dark:hover:text-white dark:focus:text-white focus:outline-hidden h-6 w-6 mx-2"

// View renders the navbar. viewAllSubs reports whether the user may see the global
// submission list; reqPath is where login/signup/logout send the user back to.
func View(ctx context.Context, viewAllSubs bool, reqPath string) Node {
	u := user.UserBriefContext(ctx)

	return Group{
		Header(Class("segment-panel my-1 text-2xl md:flex md:justify-between md:px-5 md:py-3 md:items-center"),
			Div(Class("flex items-center justify-between px-4 py-3 md:p-0"),
				Div(Class("inline-block"),
					A(Class("roboto h-8 black-anchor"), Href("/"), Text(flags.NavbarBranding.Value())),
				),
				Div(Class("md:hidden inline-block"),
					A(ID("theme_button_mobile"), Class(mobileButtonClass), Href("#"), themeIcon(ctx)),
					Button(ID("nav-toggler"), Type("button"), Attr("onclick", "navbar_mgr.toggleNavbar()"),
						Class("fas fa-bars "+mobileButtonClass), Aria("label", ui.TS(ctx, "toggleNavbar"))),
				),
			),
			Nav(ID("nav-dropdown"), Class("px-2 pt-1 pb-3 md:flex md:p-0 z-50"),
				A(ID("theme_button"), Class("hidden md:block black-anchor mt-1 md:mt-0 md:ml-1 px-2 py-1 rounded-sm hoverable"), Href("#"), themeIcon(ctx)),
				If(flags.NavbarProblems.Value(), navLink("/problems", ui.T(ctx, "problem.list"))),
				If(flags.NavbarContests.Value(), navLink("/contests/official", ui.T(ctx, "contests"))),
				If(viewAllSubs && flags.NavbarSubmissions.Value(), navLink("/submissions", ui.T(ctx, "submission.list"))),
				If(u == nil, Group{
					If(flags.SignupEnabled.Value(), navLink("/signup?back="+reqPath, ui.T(ctx, "auth.signup"))),
					navLink("/login?back="+reqPath, ui.T(ctx, "auth.login")),
				}),
				Iff(u != nil, func() Node { return profileDropdown(ctx, u, viewAllSubs, reqPath) }),
			),
		),
		Script(Raw(`var navbar_mgr = new bundled.NavBarManager();
document.getElementById("theme_button").addEventListener("click", bundled.toggleTheme);
document.getElementById("theme_button_mobile").addEventListener("click", bundled.toggleTheme);`)),
	}
}

func profileDropdown(ctx context.Context, u *kilonova.UserBrief, viewAllSubs bool, reqPath string) Node {
	divider := Div(Class("dropdown-divider"))
	return Div(ID("pr-dropdown"), Class("relative"),
		Button(Attr("onclick", "navbar_mgr.toggleDropdown()"), ID("profile-dropdown-button"),
			Class("relative z-10 block black-anchor mt-1 md:mt-0 md:ml-1 px-2 py-1 rounded-sm hoverable"),
			Text(u.Name), I(ID("dropdown-caret"), Class("ml-1 fas fa-caret-down")),
		),
		Div(ID("profile-dropdown"), Class("dropdown-list hidden"),
			dropdownItem(ctx, "/profile", "fa-user", "profile.url"),
			dropdownItem(ctx, "/settings", "fa-user-cog", "settings"),
			If(viewAllSubs && !flags.NavbarSubmissions.Value(), dropdownItem(ctx, "/submissions", "fa-table-list", "submission.list")),
			If(!flags.NavbarContests.Value(), dropdownItem(ctx, "/contests/official", "fa-medal", "contests")),
			dropdownItem(ctx, "/tags/", "fa-tag", "tags"),
			If(u.IsProposer(), Group{
				divider,
				dropdownItem(ctx, "/proposer", "fa-chalkboard-teacher", "panel.proposer"),
				dropdownItem(ctx, "/problem_lists", "fa-list-ul", "problem_lists"),
				dropdownItem(ctx, "/posts", "fa-newspaper", "blog_posts"),
			}),
			If(u.IsAdmin(), Group{
				divider,
				dropdownItem(ctx, "/admin", "fa-sliders-h", "panel.admin"),
				dropdownItem(ctx, "/admin/users", "fa-users", "users"),
				dropdownItem(ctx, "/admin/auditLog", "fa-file-medical-alt", "panel.audit_log"),
				dropdownItem(ctx, "/admin/oauth", "fa-key", "oauth_clients"),
				dropdownItem(ctx, "/admin/debug", "fa-bug-slash", "panel.debug"),
				dropdownItem(ctx, "/problems?ordering=requested_review&published=false&descending=true", "fa-magnifying-glass", "panel.review"),
				dropdownItem(ctx, "/grader", "fa-heart-pulse", "panel.grader"),
			}),
			divider,
			Form(Method("POST"), Action("/logout"), Class("dropdown-list-item"),
				Input(Type("hidden"), Name("back"), Value(reqPath)),
				Button(Type("submit"), dropdownIcon("fa-sign-out-alt"), Text(" "), ui.T(ctx, "auth.logout")),
			),
		),
	)
}

// themeIcon shows the icon for switching away from the current theme.
// session.ts swaps it client-side when the theme is toggled.
func themeIcon(ctx context.Context) Node {
	icon := "fa-moon"
	if util.Theme(ctx) == kilonova.PreferredThemeDark {
		icon = "fa-lightbulb"
	}
	return Group{
		I(Class("fas fa-fw " + icon)),
		Span(Class("fa-sr-only"), ui.T(ctx, "toggleTheme")),
	}
}

func navLink(href string, children ...Node) Node {
	return A(Class("block black-anchor mt-1 md:mt-0 md:ml-1 px-2 py-1 rounded-sm hoverable"), Href(href), Group(children))
}

func dropdownItem(ctx context.Context, href, icon, key string) Node {
	return A(Class("dropdown-list-item"), Href(href), dropdownIcon(icon), Text(" "), ui.T(ctx, key))
}

func dropdownIcon(icon string) Node {
	return I(Class("ml-n2 fas " + icon + " fa-fw"))
}
