package kilonova

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"strings"
)

const randomCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString returns a new string of a specified size containing only [a-zA-Z0-9] characters
func RandomString(size int) string {
	sb := strings.Builder{}
	sb.Grow(size)
	for ; size > 0; size-- {
		sb.WriteByte(randomCharacters[rand.Intn(len(randomCharacters))])
	}
	return sb.String()
}

func RandomSaltedString(salt string) string {
	vidB := sha256.Sum256([]byte(RandomString(16) + salt))
	return hex.EncodeToString(vidB[:])
}
