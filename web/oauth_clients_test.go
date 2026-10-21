package web

import "testing"

func TestParseRedirects(t *testing.T) {
	got, err := parseRedirects("https://a.com/cb\r\n\n  com.example.app:/cb  \n")
	if err != nil || len(got) != 2 || got[1] != "com.example.app:/cb" {
		t.Fatalf("unexpected result: %v, %v", got, err)
	}
	for _, bad := range []string{"/relative", "https://a.com/cb#frag", "not a url"} {
		if _, err := parseRedirects(bad); err == nil {
			t.Errorf("expected %q to be rejected", bad)
		}
	}
}
