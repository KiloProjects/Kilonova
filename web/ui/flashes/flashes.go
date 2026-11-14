// Package flashes is the registry of flash announcements: dismissable banners that
// web/assets/flash.ts reveals unless the visitor dismissed them. A Flash can only be
// created here, so every renderable flash is listed by All and can be hidden per
// instance through flags.HiddenFlashes.
package flashes

import (
	"context"
	"slices"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// Flash is one registered announcement. Its name is also the browser-side dismissal key.
type Flash struct {
	name        string
	description string
}

func (f Flash) Name() string        { return f.name }
func (f Flash) Description() string { return f.description }

var (
	GPC = Flash{
		name:        "gpc_announcement",
		description: "Girls' Programming Camp banner on the home page, inviting girls to qualify after the national olympiad (ONI).",
	}
	ContestDisclaimer = Flash{
		name:        "contest_disclaimer",
		description: "Warning on problem pages of problems that came from a contest, saying the statement or tests may differ from the original.",
	}
)

// All lists every registered flash, in display order.
func All() []Flash {
	return []Flash{GPC, ContestDisclaimer}
}

// IsHidden reports whether f is hidden on the instance serving ctx.
func IsHidden(ctx context.Context, f Flash) bool {
	return slices.Contains(flags.HiddenFlashes.Get(ctx), f.name)
}

// Panel renders f's dismissable section around children, or nothing when f is hidden
// on this instance. Attributes in children land on the section.
func Panel(ctx context.Context, f Flash, children ...Node) Node {
	if IsHidden(ctx, f) {
		return Group{} // not nil: callers may render it directly or put it in a slice
	}
	return Section(Data("flashname", f.name), Class("segment-panel hidden"), Group(children))
}

// DismissButton is the close button that dismisses f for this visitor.
func DismissButton(f Flash) Node {
	return Button(Data("flashdiscard", f.name), I(Class("modal-close fas fa-xmark")))
}
