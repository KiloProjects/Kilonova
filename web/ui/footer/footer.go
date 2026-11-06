// Package footer renders the site-wide page footer.
package footer

import (
	"context"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// View renders the footer. Call it where the footer goes in the page, not earlier:
// the render duration it shows is measured up to the call.
func View(ctx context.Context, prepareDuration time.Duration, renderStart time.Time) Node {
	lang := util.LanguageContext(ctx)

	return Group{
		Footer(Class("segment-panel my-1"),
			Div(Class("min-h-0 mx-auto"),
				Div(Class("grid gap-2 grid-cols-1 md:grid-cols-2 lg:grid-cols-3 mb-2 justify-between"),
					Div(Class("h-auto mb-2"),
						Div(Class("text-lg"), ui.T(ctx, "usefulInfo")),
						Ul(
							Li(link("/", ui.T(ctx, "frontPage"))),
							If(flags.DonationsEnabled.Value(), Li(link("/donate", ui.T(ctx, "donate")))),
							Li(link("/problems", ui.T(ctx, "problem.list"))),
							Li(link("/contests/official", ui.T(ctx, "contests"))),
							Li(link("/submissions", ui.T(ctx, "submission.list"))),
							Li(link("/posts", ui.T(ctx, "blogpost.list"))),
						),
						Label(Class("block my-2"),
							Select(ID("sessionLang"), Class("form-select"), AutoComplete("off"),
								Option(Value("ro"), If(lang == "ro", Selected()), Text("🇷🇴 Română")),
								Option(Value("en"), If(lang == "en", Selected()), Text("🇬🇧 English")),
							),
							Span(Text(" "), ui.T(ctx, "displayLang")),
						),
					),
					Div(Class("h-auto mb-2"),
						Div(Class("text-lg"), ui.T(ctx, "otherInfo")),
						Ul(
							Li(Span(Class("black-span"), icon("fas fa-code-branch"), Text(" Kilonova "+kilonova.Version))),
							Li(Span(Class("black-span"), icon("fas fa-clock"), Text(" "), ui.T(ctx, "serverTime"), Text(": "),
								Span(ID("footer_server_time"), Text(time.Now().Format("15:04:05"))))),
							If(kilonova.DebugMode(), Li(Span(Class("black-span"), icon("text-red-600 fas fa-exclamation-triangle"), Text(" Debug Mode active")))),
							If(flags.DiscordInviteID.Value() != "", Li(link("/links/discord", icon("fab fa-discord"), Text(" Discord Server"), Target("_blank")))),
							Li(externalLink("https://github.com/KiloProjects/Kilonova", icon("fab fa-github"), Text(" GitHub"))),
							Li(link("/termsOfService", icon("fas fa-gavel"), Text(" "), ui.T(ctx, "termsOfService"), Rel("noopener noreferrer"))),
							Li(link("/privacyPolicy", icon("fas fa-unlock"), Text(" "), ui.T(ctx, "privacyPolicy"), Rel("noopener noreferrer"))),
							Li(externalLink("mailto:admin@kilonova.ro?subject=Contact", icon("fas fa-envelope"), Text(" Contact"))),
						),
					),
					Div(Class("h-auto mb-2 reset-list"),
						Div(Class("text-lg"), ui.T(ctx, "credits")),
						Ul(
							Li(Class("pl-0"), externalLink("https://github.com/AlexVasiluta", Text("Platform made with "), icon("fas fa-heart"), Text(" by Alex Vasiluță "))),
							Li(Class("black-span pl-0"), Text("Problems generously uploaded by the Kilonova community")),
							If(flags.CCDisclaimer.Value(), Li(Class("black-anchor pl-0"),
								Text("Cu excepția cazurilor unde este specificat diferit, conținutul platformei este publicat sub licența "),
								A(Rel("license"), Href("https://creativecommons.org/licenses/by-nc-sa/4.0/"), Text("CC BY-NC-SA 4.0")),
								Text(". Vedeți "), A(Href("https://kilonova.ro/posts/creative-commons"), Text("aici")), Text(" mai multe detalii. "),
								A(Rel("license"), Href("https://creativecommons.org/licenses/by-nc-sa/4.0/"),
									Img(Alt("Creative Commons Licence"), Style("border-width:0"), Width("88"), Height("31"), Src("/static/by-nc-sa.eu.png")),
								),
							)),
						),
					),
				),
				Iff(flags.FooterTimings.Value(), func() Node {
					return Details(Class("text-muted reset-list"),
						Summary(ui.T(ctx, "requestStats")),
						Ul(
							Li(ui.T(ctx, "prepareDuration"), Text(": "+prepareDuration.String())),
							Li(ui.T(ctx, "renderDuration"), Text(": "+time.Since(renderStart).String())),
							Iff(user.UserBriefContext(ctx).IsAdmin(), func() Node {
								return Li(ui.T(ctx, "queryCount"), Text(": ~"+strconv.FormatInt(sudoapi.GetQueryCounter(ctx), 10)))
							}),
						),
					)
				}),
			),
		),
		Script(Raw(`document.getElementById("sessionLang").addEventListener("change", (e) => {
	e.preventDefault();
	bundled.setLanguage(e.currentTarget.value);
})`)),
		If(flags.GoatCounterDomain.Value() != "", Script(Data("goatcounter", flags.GoatCounterDomain.Value()+"/count"), Async(), Src(flags.GoatCounterDomain.Value()+"/count.js"))),
	}
}

func link(href string, children ...Node) Node {
	return A(Class("black-anchor"), Href(href), Group(children))
}

func externalLink(href string, children ...Node) Node {
	return link(href, Target("_blank"), Rel("noopener noreferrer"), Group(children))
}

func icon(class string) Node {
	return I(Class(class + " fa-fw"))
}
