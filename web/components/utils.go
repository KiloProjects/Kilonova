package components

import (
	"context"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func T(ctx context.Context, key string, vals ...any) string {
	return kilonova.GetText(util.LanguageContext(ctx), key, vals...)
}
