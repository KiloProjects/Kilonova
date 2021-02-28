package kilonova

import (
	"strings"
	"testing"
)

func TestRandomString(t *testing.T) {
	var sizes = []int{5, 7, 8, 32, 64} // idk test for a few random sizes

	for _, size := range sizes {
		str := RandomString(size)
		if len(str) != size {
			t.Fatalf("Wanted string of size %d, got %d", size, len(str))
		}
		for _, chr := range str {
			if !strings.ContainsRune(randomCharacters, chr) {
				t.Fatal("String contains characters other than the specified ones")
			}
		}
	}
}
