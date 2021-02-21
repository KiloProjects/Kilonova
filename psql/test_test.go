package psql

import (
	"testing"

	"github.com/matryer/is"
)

func TestTestByID(t *testing.T) {
	if !*testTest {
		t.SkipNow()
	}
	is := is.New(t)

	test, err := tserv.TestByID(ctx, 1)
	is.NoErr(err)
	t.Logf("%#v\n", test)
}

func TestTestsByPbID(t *testing.T) {
	if !*testTest {
		t.SkipNow()
	}
	is := is.New(t)

	tests, err := tserv.Tests(ctx, 1)
	is.NoErr(err)
	for _, test := range tests {
		t.Logf("%#v\n", test)
	}
}
func TestBiggestVID(t *testing.T) {
	if !*testTest {
		t.SkipNow()
	}
	is := is.New(t)

	bvid, err := tserv.BiggestVID(ctx, 1)
	is.NoErr(err)
	t.Logf("%d\b", bvid)
}
