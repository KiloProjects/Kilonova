package ui

import (
	"context"

	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Captcha renders the CAPTCHA image and its answer field, submitted with the enclosing form
// as captcha_id and captcha_response. The script only reloads the image.
func Captcha(ctx context.Context, id string) Node {
	return Group{
		h.Div(h.ID("captcha_container"), h.Class("segment-panel my-2"),
			h.P(
				T(ctx, "auth.captcha.title"), Text(":"), h.Br(),
				h.Img(h.ID("captcha_img"), h.Class("rounded-sm"), h.Src("/captcha/"+id+".png"), h.Alt("Captcha image"), h.Width("240"), h.Height("80"), h.Style("background-color: white;")),
			),
			h.Input(h.Type("hidden"), h.Name("captcha_id"), h.Value(id)),
			h.Label(h.Class("block my-2"),
				h.Span(T(ctx, "response")), Text(" "),
				h.Input(h.Type("text"), h.Name("captcha_response"), h.Class("form-input w-full"), h.Pattern("[0-9]*"), h.AutoComplete("off")),
			),
			Button(h.ID("captcha_reload_btn"), h.I(h.Class("fas fa-fw fa-rotate")), Text(" "), T(ctx, "reload")),
		),
		h.Script(Raw(`
document.getElementById("captcha_reload_btn").addEventListener("click", () => {
	const img = document.getElementById("captcha_img")
	img.src = img.src.split("?")[0] + "?reload=" + Date.now()
})
`)),
	}
}
