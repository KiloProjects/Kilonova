package kilonova

import (
	"regexp"

	"github.com/davecgh/go-spew/spew"
)

var (
	SubtaskRegex = regexp.MustCompilePOSIX("(;?[tf]?[0-9]+-[0-9]+;?)+")
)

// SubTasks are split in the following format:
// - semicolons are placed between "subtask groups".
// 		A semicolon is assumed to be automatically considered to be a part at the start and at the end of the string
// - the first character after a semicolon can be a boolean flag ('t'|'f') signifying if the tests should be run in order or not. If no flag is specified, it defaults to `true`.
//     In other words, if the flag is true, then all tests will be run regardless of correctness. If the flag is false, the first test that does not get a max score is the last one evaluated in the group.
// All tests that do not fit in a subtask group are put in a separate subtask group
// - a range `start-end` marking all grouped tests for the subtask.
// The regex for a valid subtask string is `/(;?[tf]?[0-9]+-[0-9]+;?)+/g`
// The subtask string must not have any overlapping ranges.
// The subtask string must have all ranges in ascending order.
// The remaining subtask modes run in `true` flag mode.
// An example of a correct subtask string would be: `1-3;f4-5;t6-7`.
// An invalid subtask string would be `3-5f;1-2;3-4`
type SubTasks struct {
	Groups []SubTaskGroup
}

type SubTaskGroup struct {
	Start int
	Stop  int
	DoAll bool
}

type SubTestGroup struct {
	SubTests  []*SubTest
	GroupInfo *SubTaskGroup
}

// Split returns a slice of slices
// The subtest slice must be sorted by visible ID
// Note that subtask order is not guaranteed to be maintained, while
func (s *SubTasks) Split(subtests []*SubTest) []*SubTestGroup {
	// TODO
	return nil
}

func ParseSubtaskString(subtaskString string) (*SubTasks, error) {
	// TODO
	spew.Dump(SubtaskRegex.FindStringSubmatch(subtaskString))
	return &SubTasks{}, nil
}
