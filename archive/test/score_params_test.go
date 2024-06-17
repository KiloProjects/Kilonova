package test_test

import (
	"testing"

	"github.com/KiloProjects/kilonova/archive/test"
)

type scoreParameterTest struct {
	Str       string
	NumGroups int
	Error     bool
}

var scoreParameterExamples = map[string]scoreParameterTest{
	"simple": {Str: `[[0, 2], [10, 6], [20, 6], [30, 6], [40, 8]]`, NumGroups: 5, Error: false},
	"regex":  {Str: `[[0, "98|99"], [20, "00|01|02|03|04"], [20, "05|06|07|08|09"], [60, ".*"]]`, NumGroups: 4, Error: false},
	"fail1":  {Str: `[[0, -1]]`, Error: true},
	"fail2":  {Str: `[[0, 2], [100, "1213"]]`, Error: true},
	"fail3":  {Str: `[[0, "12345"], [100, 1]]`, Error: true},
	"fail4":  {Str: `[[0, 1, 2], [3, 4, 5], [6]]`, Error: true},
	"fail5":  {Str: `[[-1, 5]]`, Error: true},
}

func TestParseScoreParameters(t *testing.T) {
	for k, v := range scoreParameterExamples {
		v := v
		t.Run(k, func(t *testing.T) {
			t.Parallel()
			val, err := test.ParseScoreParameters([]byte(v.Str))
			if err != nil && !v.Error {
				t.Fatalf("Error parsing score parameters: %#v", err)
			}
			if err == nil && v.Error {
				t.Fatalf("Test should not succeed")
			}
			if !v.Error {
				if len(val) != v.NumGroups {
					t.Fatalf("Invalid number of groups, expected %d, got %d ", v.NumGroups, len(val))
				}
				// t.Log(spew.Sdump(val))
			}
		})
	}
}
