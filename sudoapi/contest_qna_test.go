package sudoapi

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/KiloProjects/kilonova"
)

// The length check runs before any database access, so a zero BaseAPI is enough.
func TestCreateContestQuestionTooLong(t *testing.T) {
	text := strings.Repeat("ă", kilonova.MaxContestQuestionLen+1)
	_, err := (&BaseAPI{}).CreateContestQuestion(context.Background(), &kilonova.Contest{}, 1, text)
	if code := kilonova.ErrorCode(err); code != http.StatusBadRequest {
		t.Fatalf("got status %d (err %v), want 400", code, err)
	}
}
