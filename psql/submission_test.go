package psql

import (
	"fmt"
	"testing"

	"github.com/KiloProjects/kilonova"
	"github.com/matryer/is"
)

func TestSubByID(t *testing.T) {
	if !*testSubs {
		t.SkipNow()
	}
	is := is.New(t)

	sub, err := sserv.SubmissionByID(ctx, 1)
	is.NoErr(err)
	t.Logf("%d\n", sub.ID)
}

func TestSubFilters(t *testing.T) {
	if !*testSubs {
		t.SkipNow()
	}
	is := is.New(t)

	filters := []kilonova.SubmissionFilter{
		{
			Status: kilonova.StatusWaiting,
			Offset: 5,
			Limit:  3,
		},
		{
			UserID:    iPtr(1),
			ProblemID: iPtr(1),
			Limit:     2,
			Offset:    1,
		},
	}

	for i, filter := range filters {
		tName := fmt.Sprintf("filter-%d", i)
		filter := filter
		t.Run(tName, func(t *testing.T) {
			t.Parallel()
			subs, err := sserv.Submissions(ctx, filter)
			is.NoErr(err)
			for _, sub := range subs {
				t.Logf("%d\n", sub.ID)
			}
		})
	}
}
