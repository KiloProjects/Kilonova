package tasks

import (
	"testing"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/stretchr/testify/require"
)

// TestMergeStatsFirstUserFailureWins: like CMS, the first user process that
// failed decides the verdict. Being interrupted by the scheduler (SIGINT) or
// writing to a manager that already exited (SIGPIPE) is not such a failure.
func TestMergeStatsFirstUserFailureWins(t *testing.T) {
	var (
		ok          = &eval.RunStats{}
		wall        = &eval.RunStats{Status: "TO", Message: "Time limit exceeded (wall clock)"}
		crash       = &eval.RunStats{Status: "SG", ExitSignal: 11, Message: "Caught fatal signal 11"}
		nonzero     = &eval.RunStats{Status: "RE", ExitCode: 1, Message: "Exited with error status 1"}
		interrupted = &eval.RunStats{Status: "SG", ExitSignal: 2, Message: "Interrupted"}
		sigpipe     = &eval.RunStats{Status: "SG", ExitSignal: 13, Message: "Caught fatal signal 13"}
	)
	for name, tc := range map[string]struct {
		stats []*eval.RunStats
		want  string
	}{
		"all fine":                       {[]*eval.RunStats{ok, ok}, ""},
		"wall timeout":                   {[]*eval.RunStats{ok, wall}, wall.Message},
		"crash":                          {[]*eval.RunStats{crash, ok}, crash.Message},
		"first failure wins":             {[]*eval.RunStats{nonzero, wall}, nonzero.Message},
		"stopped by scheduler":           {[]*eval.RunStats{interrupted, ok}, ""},
		"manager hung up":                {[]*eval.RunStats{sigpipe, interrupted}, ""},
		"failure after run-over endings": {[]*eval.RunStats{sigpipe, interrupted, crash}, crash.Message},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, tc.want, mergeStats(true, tc.stats...).Message)
		})
	}

	oom := &eval.RunStats{MemoryLimitExceeded: true}
	require.True(t, mergeStats(true, ok, oom).MemoryLimitExceeded)
}
