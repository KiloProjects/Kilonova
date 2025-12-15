package tutils

import (
	"context"
	"io"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/a-h/templ"
)

func T(ctx context.Context, key string, vals ...any) string {
	return kilonova.GetText(util.LanguageContext(ctx), key, vals...)
}

func TC(key string, vals ...any) templ.ComponentFunc {
	return func(ctx context.Context, w io.Writer) error {
		_, err := io.WriteString(w, templ.EscapeString(kilonova.GetText(util.LanguageContext(ctx), key, vals...)))
		return err
	}
}
