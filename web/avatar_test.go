package web

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAvatarSize(t *testing.T) {
	for in, want := range map[string]int{"": 128, "x": 128, "0": 128, "-5": 16, "1": 16, "32": 32, "33": 64, "128": 128, "512": 512, "99999": 512} {
		assert.Equal(t, want, avatarSize(in), "avatarSize(%q)", in)
	}
}
