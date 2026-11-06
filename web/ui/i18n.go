package ui

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

// TS returns the translation of key in the request's language as a plain string,
// for attribute values such as aria-label or placeholder.
func TS(ctx context.Context, key string, vals ...any) string {
	return kilonova.GetText(util.LanguageContext(ctx), key, vals...)
}

// T renders the translation of key in the request's language, escaped.
func T(ctx context.Context, key string, vals ...any) Node {
	return Text(TS(ctx, key, vals...))
}

// TRaw renders the translation of key unescaped, for translations that contain markup.
// The translation is trusted, but vals are not escaped: never pass user input as vals.
func TRaw(ctx context.Context, key string, vals ...any) Node {
	return Raw(TS(ctx, key, vals...))
}
