// Package layout renders the full HTML document every page is wrapped in.
package layout

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui"
	"github.com/KiloProjects/kilonova/web/ui/footer"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/components"
	. "maragu.dev/gomponents/html"
)

type middlewareStartCtx string

// MiddlewareStartKey holds the time.Time the request entered the middleware chain.
const MiddlewareStartKey = middlewareStartCtx("middleware_start")

// Assets resolves a source path from web/assets to the hashed file Vite built.
type Assets interface {
	Asset(ctx context.Context, src string) string
}

type Props struct {
	EnabledLanguages map[string]string
	Title            string
	Description      string
	Navbar           Node // nil renders no navbar
	Head             Node
	Content          Node
	Assets           Assets
	HideFooter       bool
}

func View(ctx context.Context, p Props) Node {
	renderStart := time.Now()
	if p.Title == "" {
		p.Title = "Kilonova"
	}
	if p.Description == "" {
		p.Description = "Modern competitive programming platform"
	}
	if p.Content == nil {
		p.Content = H1(Text("Well... this is awkward... There's no content"))
	}

	return HTML5(HTML5Props{
		Title:       p.Title,
		Description: p.Description,
		Language:    util.LanguageContext(ctx),
		HTMLAttrs:   Group{If(util.Theme(ctx) != "", Class(string(util.Theme(ctx))))},
		Head:        Group{headTags(ctx, p), p.Head},
		Body: Group{Attr("hx-ext", "morph"),
			p.Navbar,
			Iff(user.UserFullContext(ctx) != nil && !user.UserFullContext(ctx).VerifiedEmail, func() Node {
				return Div(Class("w-full py-1 px-2 bg-green-200 dark:bg-green-700 text-black dark:text-white text-center"),
					ui.TRaw(ctx, "unverifiedEmail"))
			}),
			Main(Class("c-container"), p.Content),
			Div(ID("modals")),
			// Built at render time, after the content, so the footer's render duration and
			// query count cover the whole page.
			If(!p.HideFooter, NodeFunc(func(w io.Writer) error {
				start, _ := ctx.Value(MiddlewareStartKey).(time.Time)
				return footer.View(ctx, renderStart.Sub(start), renderStart).Render(w)
			})),
		},
	})
}

func headTags(ctx context.Context, p Props) Node {
	return Group{
		Link(Rel("apple-touch-icon"), Attr("sizes", "180x180"), Href("/static/favicons/favicon.ico")),
		Link(Rel("icon"), Type("image/png"), Attr("sizes", "32x32"), Href("/static/favicons/favicon-32x32.png")),
		Link(Rel("icon"), Type("image/png"), Attr("sizes", "16x16"), Href("/static/favicons/favicon-16x16.png")),
		Link(Rel("manifest"), Href("/static/favicons/site.webmanifest?v=4")),
		Link(Rel("mask-icon"), Href("/static/favicons/safari-pinned-tab.svg?v=4"), Attr("color", "#5bbad5")),
		Link(Rel("shortcut icon"), Href("/static/favicons/favicon.ico")),
		Meta(Name("apple-mobile-web-app-title"), Content("Kilonova")),
		Meta(Name("application-name"), Content("Kilonova")),
		Meta(Name("msapplication-TileColor"), Content("#da532c")),
		Meta(Name("msapplication-config"), Content("/static/favicons/browserconfig.xml?v=4")),
		Meta(Name("theme-color"), Content("#ffffff")),
		Meta(Attr("property", "og:title"), Content(p.Title)),
		Meta(Attr("property", "og:type"), Content("website")),
		Meta(Attr("property", "og:description"), Content(p.Description)),
		Meta(Attr("property", "og:locale"), Content("ro_RO")),
		Meta(Attr("property", "og:site_name"), Content("Kilonova")),
		Meta(Attr("property", "og:image"), Content("/static/ogimage.jpeg")),

		jsonScript("server_time", time.Now().Format(time.RFC3339Nano)),
		jsonScript("platform_info", platformInfo(ctx, p.EnabledLanguages)),
		Script(Raw(`window.platform_info = JSON.parse(document.getElementById("platform_info").textContent);`)),
		Script(Src(p.Assets.Asset(ctx, "app.ts"))),
		Script(Src(p.Assets.Asset(ctx, "vendored.ts"))),
		Script(Raw(`const server_time = JSON.parse(document.getElementById("server_time").textContent);
bundled.initTime(bundled.dayjs(), bundled.dayjs(server_time));
window.htmx.on("htmx:afterSwap", (el) => {
	bundled.initialLoad(el.target);
});
if (platform_info.debug) {
	window.htmx.logAll();
}`)),
		Link(Rel("stylesheet"), Href(p.Assets.Asset(ctx, "chroma.css"))),
		Link(Rel("stylesheet"), Href(p.Assets.Asset(ctx, "vendor.css"))),
		Link(Rel("stylesheet"), Href(p.Assets.Asset(ctx, "tailwind.css"))),
	}
}

// jsonScript embeds data for the frontend to JSON.parse. encoding/json escapes <, > and &,
// so the data can't close the script tag.
func jsonScript(id string, data any) Node {
	return NodeFunc(func(w io.Writer) error {
		b, err := json.Marshal(data)
		if err != nil {
			return err
		}
		return Script(ID(id), Type("application/json"), Raw(string(b))).Render(w)
	})
}
