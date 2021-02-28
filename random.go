package kilonova

import (
	"math/rand"
	"strings"
)

const randomCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

// RandomString returns a new string of a specified size containing only [a-zA-Z0-9_-] characters
func RandomString(size int) string {
	sb := strings.Builder{}
	sb.Grow(size)
	for ; size > 0; size-- {
		sb.WriteByte(randomCharacters[rand.Intn(len(randomCharacters))])
	}
	return sb.String()
}
