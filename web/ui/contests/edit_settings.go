package contests

import (
	"context"
	"fmt"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

// settingsForm is the main contest settings form: general options, limits and leaderboard.
func settingsForm(ctx context.Context, p EditParams) Node {
	c := p.Contest
	return Form(Class("segment-panel"), ID("contest_update_form"), AutoComplete("off"),
		H2(ui.T(ctx, "header.contest.main")),
		Div(Class("grid grid-cols-1 lg:grid-cols-2"),
			generalSettings(ctx, c, p.IsAdmin),
			limitSettings(ctx, c, p.IsAdmin),
			leaderboardSettings(ctx, c),
		),
		Div(
			ui.SubmitButton(ui.T(ctx, "button.update")),
			Text(" "),
			ui.Button(ID("cloneContestButton"), Class("ml-2"), ui.T(ctx, "cloneContest")),
			If(!c.Started(), Div(Class("block my-2"),
				ui.DangerButton(ID("deleteContestButton"), Class("mr-2"), ui.T(ctx, "deleteContest")),
			)),
		),
	)
}

func generalSettings(ctx context.Context, c *kilonova.Contest, admin bool) Node {
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "header.contest.general")),
		textField(ctx, "name", Input(Class("form-input"), ID("contest_name_upd"), Name("name"), Type("text"), Value(c.Name), Required())),
		textField(ctx, "startTime", Input(Class("form-input"), ID("start_time"), Name("start_time"), Type("datetime-local"), Required())),
		textField(ctx, "endTime", Input(Class("form-input"), ID("end_time"), Name("end_time"), Type("datetime-local"), Required())),
		Label(Class("block my-2"),
			Span(Classes{"form-label": true, "text-muted": !admin}, ui.T(ctx, "contest_type"), Text(": ")),
			Text(" "),
			Select(ID("contest_type"), Type("text"), Classes{"form-select": true, "text-muted": !admin}, Value("virtual"), If(!admin, Disabled()), AutoComplete("off"),
				Option(Value("official"), If(c.Type == kilonova.ContestTypeOfficial, Selected()), ui.T(ctx, "official")),
				Option(Value("virtual"), If(c.Type == kilonova.ContestTypeVirtual, Selected()), ui.T(ctx, "virtual")),
			),
			P(Class("block text-muted text-sm"), ui.T(ctx, "contest_type_warn")),
		),
		checkbox("c_public_join", "public_join", c.PublicJoin, false, ui.T(ctx, "publicJoin")),
		checkbox("c_reg", "register_during_contest", c.RegisterDuringContest, false, Group{
			ui.T(ctx, "register_during_contest"), Text(" "),
			Span(Class("text-sm text-muted"), Text(" ("), ui.T(ctx, "register_during_contest_warning"), Text(")")),
		}),
		checkbox("c_visible", "visible", c.Visible, false, ui.T(ctx, "visible")),
		Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, "per_user_time"), Text(": ")),
			Text(" "),
			Input(Class("form-input"), Name("per_user_time"), Type("number"), Value(strconv.Itoa(c.PerUserTime)), Required()),
			Text(" "),
			Span(Class("form-label"), ui.T(ctx, "seconds")),
			P(Class("text-sm text-muted"), ui.T(ctx, "per_user_time_warning")),
		),
	)
}

func limitSettings(ctx context.Context, c *kilonova.Contest, admin bool) Node {
	seconds := func(key, name string, extra ...Node) Node {
		return Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, key), Text(": ")),
			Text(" "),
			Input(Class("form-input"), Name(name), Type("number"), Group(extra), Required()),
			Text(" "),
			Span(Class("form-label"), ui.T(ctx, "seconds")),
		)
	}
	return Div(Class("segment-panel"),
		H2(ui.T(ctx, "header.contest.limits")),
		textField(ctx, "maxSubs", Input(Class("form-input"), Name("max_subs"), Type("number"), Value(strconv.Itoa(c.MaxSubs)), Required())),
		seconds("questionCooldown", "question_cooldown", Min("0"), Value(fmt.Sprint(c.QuestionCooldown.Seconds()))),
		seconds("submissionCooldown", "submission_cooldown", Min("0"), Step("1"), Value(fmt.Sprint(c.SubmissionCooldown.Seconds()))),
		checkbox("c_ip_mgmt", "ip_management_enabled", c.IPManagementEnabled, !admin, ui.T(ctx, "ip_management_enabled")),
		checkbox("c_whitelist", "whitelist_enabled", c.WhitelistEnabled, !admin, ui.T(ctx, "whitelist_enabled")),
		P(Class("block text-muted text-sm"), ui.T(ctx, "whitelist_enabled_explanation")),
	)
}

func leaderboardSettings(ctx context.Context, c *kilonova.Contest) Node {
	icpc := c.LeaderboardStyle == kilonova.LeaderboardTypeICPC
	return Div(Class("segment-panel lg:col-span-2"),
		H2(ui.T(ctx, "header.contest.leaderboard")),
		Label(Class("block my-2"),
			Span(Class("form-label"), ui.T(ctx, "leaderboard.style"), Text(":")),
			Text(" "),
			Select(Name("leaderboard_style"), ID("leaderboard_style"), Class("form-select"), Required(),
				Option(Value("classic"), Attr("default"), If(c.LeaderboardStyle == kilonova.LeaderboardTypeClassic, Selected()), ui.T(ctx, "leaderboard.classic")),
				Option(Value("acm-icpc"), If(icpc, Selected()), ui.T(ctx, "leaderboard.icpc")),
			),
		),
		Label(ID("icpcPenaltyLabel"), Classes{"block": true, "mb-2": true, "hidden": !icpc},
			Span(Class("form-label"), ui.T(ctx, "icpc_submission_penalty"), Text(": ")),
			Text(" "),
			Input(Class("form-input"), Name("icpc_submission_penalty"), Type("number"), Value(strconv.Itoa(c.ICPCSubmissionPenalty)), Required()),
		),
		checkbox("c_public_leaderboard", "public_leaderboard", c.PublicLeaderboard, false, ui.T(ctx, "public_leaderboard")),
		checkbox("c_freeze_enabled", "freeze_enabled", c.LeaderboardFreeze != nil, false, ui.T(ctx, "enable_freeze")),
		Div(Class("block mb-2"),
			Label(ID("freezeLabel"), Classes{"block": true, "mb-2": true, "hidden": c.LeaderboardFreeze == nil},
				Span(Class("form-label"), ui.T(ctx, "freeze_time"), Text(": ")),
				Text(" "),
				Input(Class("form-input"), ID("freeze_time"), Name("freeze_time"), Type("datetime-local")),
			),
		),
		checkbox("c_advanced_filters", "advanced_filters", c.LeaderboardAdvancedFilter, false, ui.T(ctx, "advanced_filters")),
	)
}

// textField is a block label: "<key>: " followed by input.
func textField(ctx context.Context, key string, input Node) Node {
	return Label(Class("block mb-2"),
		Span(Class("form-label"), ui.T(ctx, key), Text(": ")),
		Text(" "),
		input,
	)
}

// checkbox is a labelled checkbox; disabled ones get a muted label.
func checkbox(id, name string, checked, disabled bool, label Node) Node {
	return Div(Class("block mb-2"),
		Label(Class("inline-flex items-center text-lg"),
			Input(Class("form-checkbox"), ID(id), Name(name), Type("checkbox"), If(checked, Checked()), If(disabled, Disabled())),
			Span(Classes{"ml-2": true, "text-muted": disabled}, label),
		),
	)
}
