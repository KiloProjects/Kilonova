package ui

import (
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestScore(t *testing.T) {
	dec := func(s string) *decimal.Decimal { d := decimal.RequireFromString(s); return &d }
	ioi := &kilonova.Problem{ScoringStrategy: kilonova.ScoringTypeMaxSub, ScorePrecision: 2}
	icpc := &kilonova.Problem{ScoringStrategy: kilonova.ScoringTypeICPC}
	for _, c := range []struct {
		name  string
		pb    *kilonova.Problem
		score *decimal.Decimal
		unit  bool
		want  string
	}{
		{"no score", ioi, nil, true, "-"},
		{"negative is no score", ioi, dec("-1"), false, "-"},
		{"trims zeros and adds unit", ioi, dec("12.5"), true, "12.5p"},
		{"whole number", ioi, dec("100"), false, "100"},
		{"icpc solved", icpc, dec("100"), false, `<i class="fas fa-fw fa-check"></i>`},
		{"icpc unsolved", icpc, dec("40"), true, `<i class="fas fa-fw fa-xmark"></i>`},
	} {
		assert.Equal(t, c.want, uitest.Render(t, Score(c.pb, c.score, c.unit)), c.name)
	}
}
