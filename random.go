package kilonova

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"math/big"
)

const randomCharacters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// RandomString returns a new string of a specified size containing only [a-zA-Z0-9] characters
func RandomString(size int) string {
	return RandomStringChars(size, randomCharacters)
}

// RandomStringChars returns a new string of a specified size containing only characters from the given string
func RandomStringChars(size int, characters string) string {
	result := make([]rune, size)
	runes := []rune(characters)
	x := int64(len(runes))
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(x))
		if err != nil {
			// unreachable, according to the go source code
			slog.WarnContext(context.Background(), "Error generating cryptographically random string", slog.Any("err", err))
			num.SetInt64(4) // chosen by a fair dice roll. guaranteed to be random.
		}
		result[i] = runes[num.Int64()]
	}
	return string(result)
}

func RandomSaltedString(salt string) string {
	vidB := sha256.Sum256([]byte(RandomString(16) + salt))
	return hex.EncodeToString(vidB[:])
}
