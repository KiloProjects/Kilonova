package contests

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
)

func enCtx() context.Context { return context.WithValue(context.Background(), util.LangKey, "en") }

func running(id int) *kilonova.Contest {
	return &kilonova.Contest{ID: id, Name: "<c>", Type: kilonova.ContestTypeOfficial,
		StartTime: time.Now().Add(-time.Hour), EndTime: time.Now().Add(time.Hour)}
}

func TestTabs(t *testing.T) {
	ctx := enCtx()
	t.Run("anonymous sees no personal or create tab", func(t *testing.T) {
		got := uitest.Render(t, Tabs(ctx, TabsParams{Page: "/contests/virtual", CanCreate: true}))
		uitest.Contains(t, got, `<a class="p-1 topbar-selected" href="/contests/virtual">`, `<a class="p-1" href="/contests">`)
		uitest.NotContains(t, got, "/contests/personal", "/contests/create")
	})
	t.Run("authed creator sees both", func(t *testing.T) {
		got := uitest.Render(t, Tabs(ctx, TabsParams{Page: "create", Authed: true, CanCreate: true}))
		uitest.Contains(t, got, `href="/contests/personal"`, `<a class="p-1 topbar-selected" href="/contests/create">`)
	})
	t.Run("authed non-creator sees personal only", func(t *testing.T) {
		got := uitest.Render(t, Tabs(ctx, TabsParams{Authed: true}))
		uitest.Contains(t, got, `href="/contests/personal"`)
		uitest.NotContains(t, got, "/contests/create")
	})
}

func TestIndex(t *testing.T) {
	ctx := enCtx()
	uitest.Contains(t, uitest.Render(t, Index(ctx, IndexParams{})), "col-span-full")
	got := uitest.Render(t, Index(ctx, IndexParams{Contests: []Info{{Contest: running(3)}}}))
	uitest.Contains(t, got, `href="/contests/3"`)
	uitest.NotContains(t, got, "col-span-full")
}

func TestView(t *testing.T) {
	ctx := enCtx()
	t.Run("problems listed with contest links", func(t *testing.T) {
		c := running(4)
		c.Description = "x"
		got := uitest.Render(t, View(ctx, ViewParams{Topbar: tb(c, "contest_general"), DescriptionHTML: "<b>desc</b>", Sidebar: SidebarParams{Info: Info{Contest: c},
			Problems: []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 7, Name: "P"}}}}}))
		uitest.Contains(t, got, "<b>desc</b>", `href="/contests/4/problems/7"`)
		uitest.NotContains(t, got, "(#7)")
	})
	t.Run("unregistered in running contest", func(t *testing.T) {
		got := uitest.Render(t, viewProblems(ctx, SidebarParams{Info: Info{Contest: running(4)}}))
		uitest.Contains(t, got, kilonova.GetText("en", "contest_problems_participants"))
	})
	t.Run("usaco not started shows duration", func(t *testing.T) {
		c := running(4)
		c.PerUserTime = 3600
		got := uitest.Render(t, viewProblems(ctx, SidebarParams{Info: Info{Contest: c, Registration: &kilonova.ContestRegistration{}}}))
		uitest.Contains(t, got, "1h0m0s")
	})
}

func TestCommunication(t *testing.T) {
	ctx := enCtx()
	c := running(5)
	t.Run("contestant", func(t *testing.T) {
		got := uitest.Render(t, Communication(ctx, CommunicationParams{PageParams: PageParams{Topbar: tb(c, "contest_communication"), Contest: c}, CanSubmit: true,
			Announcements: []*kilonova.ContestAnnouncement{}, UserQuestions: []*kilonova.ContestQuestion{}}))
		uitest.Contains(t, got, `canedit="false"`, `id="question_submit_form"`, `encoded="W10="`, `"/contest/5/askQuestion"`)
		uitest.NotContains(t, got, "announcement_create_form\" class", "kn-question-mgr")
	})
	t.Run("editor", func(t *testing.T) {
		got := uitest.Render(t, Communication(ctx, CommunicationParams{PageParams: PageParams{Topbar: tb(c, "contest_communication"), Contest: c, IsEditor: true}}))
		uitest.Contains(t, got, `canedit="true"`, `id="announcement_create_form"`, "kn-question-mgr")
		uitest.NotContains(t, got, `id="question_submit_form"`)
	})
}

func TestInvite(t *testing.T) {
	ctx := enCtx()
	c := running(6)
	t.Run("valid invite escapes names and JS", func(t *testing.T) {
		got := uitest.Render(t, Invite(ctx, InviteParams{Contest: c, Invite: &kilonova.ContestInvitation{ID: `a"</script>`},
			Inviter: &kilonova.UserBrief{Name: "<u>"}}))
		uitest.Contains(t, got, "&lt;c&gt;", "&lt;u&gt;", `invite_id: "a\"\u003c/script\u003e"`, `"/contests/6"`, `type="running"`)
		uitest.NotContains(t, got, "<c>", "<u>", "a\"</script>")
	})
	t.Run("already registered", func(t *testing.T) {
		got := uitest.Render(t, Invite(ctx, InviteParams{Contest: c, Invite: &kilonova.ContestInvitation{ID: "x"}, AlreadyRegistered: true}))
		uitest.Contains(t, got, `<a href="/contests/6">`)
		uitest.NotContains(t, got, "acceptButton")
	})
}

func TestEdit(t *testing.T) {
	ctx := enCtx()
	c := running(8)
	c.LeaderboardStyle = kilonova.LeaderboardTypeICPC
	start := c.StartTime
	maxCnt := 1
	p := EditParams{Topbar: tb(c, "contest_edit"), Contest: c,
		Problems:    []*kilonova.ScoredProblem{{Problem: kilonova.Problem{ID: 1, Name: "A"}}, {Problem: kilonova.Problem{ID: 2}}},
		Invitations: []*kilonova.ContestInvitation{{ID: "inv", CreatorID: new(3), MaxCount: &maxCnt, RedeemCount: 1}},
		Inviters:    map[int]*kilonova.UserBrief{3: {Name: "bob"}},
		MOSS:        []*kilonova.MOSSSubmission{{ID: 1, ProblemID: 1, Language: "cpp"}, {ID: 2, ProblemID: 9, URL: "https://moss/x"}},
	}
	t.Run("non-admin", func(t *testing.T) {
		got := uitest.Render(t, Edit(ctx, p))
		uitest.Contains(t, got, `value="1,2"`, `const contest_id = 8;`, `"/contest/8/createInvitation"`,
			`<label id="icpcPenaltyLabel" class="block mb-2">`, `id="freezeLabel" class="block hidden mb-2"`,
			`document.getElementById("freeze_time").value = "";`, `setDatetime("start_time", "`+start.Format("2006-01-02T15:04:05"),
			`<s>inv</s>`, `1 / 1`, `href="/profile/bob"`, `onclick="copyInvitationURL(&#34;inv&#34;)"`,
			`<a href="/problems/1">A</a> - cpp`, `<a href="/problems/9">???</a>`, `href="https://moss/x"`,
			"${contest_id}", `id="contest_type" type="text" class="form-select text-muted"`)
		uitest.NotContains(t, got, `id="force_register_form"`, `id="contest_pblist_form"`, `id="deleteContestButton"`, `onclick="invalidateInvite(`)
	})
	t.Run("admin proposer, virtual, not started", func(t *testing.T) {
		q := p
		cc := *c
		cc.Type = kilonova.ContestTypeVirtual
		cc.StartTime = time.Now().Add(time.Hour)
		q.Contest, q.IsAdmin, q.IsProposer, q.Invitations = &cc, true, true, nil
		got := uitest.Render(t, Edit(ctx, q))
		uitest.Contains(t, got, `id="force_register_form"`, "/contest/8/forceRegister", `id="contest_pblist_form"`, `id="deleteContestButton"`)
		uitest.NotContains(t, got, `onclick="createMOSS()"`, "kn-table", `class="form-select text-muted"`)
	})
}

func tb(c *kilonova.Contest, page string) *problems.TopbarParams {
	return &problems.TopbarParams{Contest: c, Page: page}
}
