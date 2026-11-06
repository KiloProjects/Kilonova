package ui

import (
	"context"
	"strings"

	"github.com/KiloProjects/kilonova"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// StatementVariantName labels a statement variant, e.g. "🇬🇧 English - Short - Markdown".
func StatementVariantName(ctx context.Context, v *kilonova.StatementVariant) string {
	var b strings.Builder
	switch v.Language {
	case "en":
		b.WriteString("🇬🇧 English")
	case "ro":
		b.WriteString("🇷🇴 Română")
	case "hu":
		b.WriteString("🇭🇺 Magyar")
	case "de":
		b.WriteString("🇩🇪 Deutsch")
	default:
		b.WriteString(v.Language)
	}

	if len(v.Type) > 0 {
		b.WriteString(" - ")
		switch v.Type {
		case "llm", "short", "long", "editorial":
			b.WriteString(TS(ctx, "stmt_type."+v.Type))
		default:
			b.WriteString(cases.Title(language.English).String(v.Type))
		}
	}

	b.WriteString(" - ")
	switch v.Format {
	case "pdf":
		b.WriteString("PDF")
	case "md":
		b.WriteString("Markdown")
	case "tex":
		b.WriteString("LaTeX")
	default:
		b.WriteString(v.Format)
	}
	return b.String()
}
