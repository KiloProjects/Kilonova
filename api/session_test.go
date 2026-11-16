package api

import (
	"net/http/httptest"
	"testing"
)

func TestSameOrigin(t *testing.T) {
	for _, tc := range []struct {
		fetchSite, origin string
		want              bool
	}{
		{"same-origin", "", true},
		{"none", "", true},
		{"same-site", "", false},
		{"cross-site", "", false},
		{"cross-site", "https://kn.example", false}, // Sec-Fetch-Site wins over Origin
		{"", "", true},                              // curl and friends
		{"", "https://kn.example", true},
		{"", "https://evil.example", false},
		{"", "null", false},
	} {
		r := httptest.NewRequest("POST", "https://kn.example/api/user/self/setBio", nil)
		if tc.fetchSite != "" {
			r.Header.Set("Sec-Fetch-Site", tc.fetchSite)
		}
		if tc.origin != "" {
			r.Header.Set("Origin", tc.origin)
		}
		if got := sameOrigin(r); got != tc.want {
			t.Errorf("sameOrigin(Sec-Fetch-Site=%q, Origin=%q) = %v, want %v", tc.fetchSite, tc.origin, got, tc.want)
		}
	}
}
