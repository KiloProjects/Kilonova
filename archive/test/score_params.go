package test

import (
	"encoding/json"
	"regexp"

	"github.com/KiloProjects/kilonova"
	"github.com/shopspring/decimal"
)

// Score parameter documentation is located here: https://cms.readthedocs.io/en/latest/Score%20types.html#groupmin

type ScoreParamEntry struct {
	Score decimal.Decimal

	Count *int
	Match *regexp.Regexp
}

func (p ScoreParamEntry) Valid() bool {
	// At most one is nil
	// Probably can just do an xor but meh
	return (p.Count != nil || p.Match != nil) && (p.Count == nil || p.Match == nil)
}

func (p *ScoreParamEntry) UnmarshalJSON(input []byte) error {
	var args []any
	if err := json.Unmarshal(input, &args); err != nil {
		return err
	}
	if len(args) != 2 {
		return kilonova.Statusf(400, "invalid number of elements in score parameter entry: got %d, expected 2", len(args))
	}

	switch v := args[0].(type) {
	case float64:
		p.Score = decimal.NewFromFloat(v)
		if p.Score.IsNegative() {
			return kilonova.Statusf(400, "cannot have negative score")
		}
	default:
		return kilonova.Statusf(400, "invalid type %T for score", v)
	}

	switch v := args[1].(type) {
	case float64: // Case 1
		val := int(v)
		p.Count = &val
		if val <= 0 {
			return kilonova.Statusf(400, "cannot have <= 0 test count")
		}
	case string: // Case 2
		expr, err := regexp.Compile(v)
		if err != nil {
			return kilonova.Statusf(400, "invalid regular expression (%q): %#v", v, err)
		}
		p.Match = expr
	default:
		return kilonova.Statusf(400, "invalid type %T for second entry", v)
	}

	return nil
}

func ParseScoreParameters(params []byte) ([]ScoreParamEntry, *kilonova.StatusError) {
	var sParams []ScoreParamEntry

	if err := json.Unmarshal(params, &sParams); err != nil {
		return nil, kilonova.WrapError(err, "Couldn't parse score parameters")
	}

	if len(sParams) == 0 {
		return sParams, nil
	}

	pType := 1
	if sParams[0].Match != nil {
		pType = 2
	}

	for _, param := range sParams {
		if (pType == 1 && param.Match != nil) || (pType == 2 && param.Count != nil) {
			return nil, kilonova.Statusf(400, "score parameters: there must be ONLY integers or ONLY regexes, do not mix them")
		}
	}

	return sParams, nil
}

// isMaskedScoring decides whether the score parameters are actually just test scores
func isMaskedScoring(params []ScoreParamEntry, tests []archiveTest) bool {
	if len(params) == 0 {
		return false
	}
	for _, param := range params {
		if param.Count != nil {
			if *param.Count > 1 {
				return false
			}
			continue
		}
		cnt := 0
		for _, test := range tests {
			if test.Matches(param.Match) {
				cnt++
				if cnt > 1 {
					return false
				}
			}
		}
	}
	return true
}

func buildParamTestScores(aCtx *ArchiveCtx, tests []archiveTest) {
	if aCtx.scoreParameters[0].Count != nil {
		for i, param := range aCtx.scoreParameters {
			if i >= len(tests) {
				return
			}
			aCtx.testScores[tests[i].VisibleID] = param.Score
		}
		return
	}
	for _, param := range aCtx.scoreParameters {
		for _, test := range tests {
			if test.Matches(param.Match) {
				aCtx.testScores[test.VisibleID] = param.Score
			}
		}
	}
}
