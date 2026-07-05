package tutils

import (
	"context"
	"io"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/a-h/templ"
	"github.com/shopspring/decimal"
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

func RemoveTrailingZeros(score string) string {
	if !strings.ContainsRune(score, '.') {
		return score
	}
	return strings.TrimSuffix(strings.TrimRight(score, "0"), ".")
}

func ScoredProblemMaxScore(pb *kilonova.ScoredProblem, summaryDisplay bool) string {
	if pb.ScoreUserID == nil {
		return ""
	}
	if pb.MaxScore == nil || pb.MaxScore.IsNegative() {
		return "-"
	}
	if pb.ScoringStrategy == kilonova.ScoringTypeICPC {
		if pb.MaxScore.Equal(decimal.NewFromInt(100)) {
			return `<i class="fas fa-fw fa-check"></i>`
		}
		return `<i class="fas fa-fw fa-xmark"></i>`
	}
	val := RemoveTrailingZeros(pb.MaxScore.StringFixed(pb.ScorePrecision))
	if summaryDisplay {
		// Add the unit at the end
		val += "p"
	}
	return val
}
