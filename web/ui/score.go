package ui

import (
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Score shows a score for problem: "-" when there is none, a check or cross for ICPC
// problems, otherwise the number at the problem's precision, suffixed with "p" if withUnit.
func Score(problem *kilonova.Problem, score *decimal.Decimal, withUnit bool) Node {
	switch {
	case score == nil || score.IsNegative():
		return Text("-")
	case problem.ScoringStrategy == kilonova.ScoringTypeICPC && score.Equal(decimal.NewFromInt(100)):
		return h.I(h.Class("fas fa-fw fa-check"))
	case problem.ScoringStrategy == kilonova.ScoringTypeICPC:
		return h.I(h.Class("fas fa-fw fa-xmark"))
	}
	s := RemoveTrailingZeros(score.StringFixed(problem.ScorePrecision))
	if withUnit {
		s += "p"
	}
	return Text(s)
}

// RemoveTrailingZeros turns "12.500" into "12.5" and "12.000" into "12".
func RemoveTrailingZeros(score string) string {
	if !strings.ContainsRune(score, '.') {
		return score
	}
	return strings.TrimSuffix(strings.TrimRight(score, "0"), ".")
}
