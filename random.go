package kilonova

import (
	"math/rand"
	"strings"
)

const characters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-"

func RandomString(size int) string {
	sb := strings.Builder{}
	sb.Grow(size)
	for ; size > 0; size-- {
		sb.WriteByte(characters[rand.Intn(len(characters))])
	}
	return sb.String()
}
