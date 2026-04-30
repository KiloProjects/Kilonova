package user

import (
	"regexp"

	"github.com/KiloProjects/kilonova"
)

var usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func ValidUsername(name string) error {
	if !usernameRegex.MatchString(name) {
		return kilonova.Statusf(400, "Username must contain only letters, digits, underlines, dashes and dots.")
	}
	if !(len(name) >= 3 && len(name) <= 24) {
		return kilonova.Statusf(400, "Username must be between 3 and 24 characters long.")
	}
	return nil
}
