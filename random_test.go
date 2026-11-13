package kilonova

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRandomString(t *testing.T) {
	var sizes = []int{5, 7, 8, 32, 64} // test for a few random sizes

	for _, size := range sizes {
		str := RandomString(size)
		require.Len(t, str, size)
		for _, chr := range str {
			require.Contains(t, randomCharacters, string(chr), "String contains characters other than the specified ones")
		}
	}
}
