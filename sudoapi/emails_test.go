package sudoapi

import (
	"github.com/KiloProjects/kilonova"
	"strings"
	"testing"
)

func TestLinkEmail(t *testing.T) {
	for _, kind := range []string{"password_recovery", "verification"} {
		for _, lang := range []string{"en", "ro"} {
			url := "https://kn.example/x/abc?a=1&b=2"
			msg, err := linkEmail{Kind: kind, Lang: lang, Name: "<b>x</b>", URL: url, Host: "https://kn.example", Branding: "KN"}.message("a@b.c")
			if err != nil {
				t.Fatalf("%s/%s: %v", kind, lang, err)
			}
			// A missing translation comes back as its own key
			all := msg.Subject + msg.PlainContent + msg.HTMLContent
			for _, key := range []string{"mail.greeting", "mail.signature", "." + kind, "%!"} {
				if strings.Contains(all, key) {
					t.Errorf("%s/%s has an untranslated or badly formatted string (%q): %s", kind, lang, key, all)
				}
			}
			if !strings.Contains(msg.PlainContent, url) || !strings.Contains(msg.PlainContent, "<b>x</b>") || !strings.Contains(msg.PlainContent, "KN\nhttps://kn.example") {
				t.Errorf("%s/%s plain is missing the link, the name or the signature: %s", kind, lang, msg.PlainContent)
			}
			if !strings.Contains(msg.HTMLContent, `<a href="https://kn.example/x/abc?a=1&amp;b=2">`) || strings.Contains(msg.HTMLContent, "style=") || !strings.Contains(msg.HTMLContent, "&lt;b&gt;x&lt;/b&gt;") || strings.Contains(msg.HTMLContent, "<b>") {
				t.Errorf("%s/%s html is missing the link or does not escape the name: %s", kind, lang, msg.HTMLContent)
			}
		}
	}
}

func TestGeneratedUserEmail(t *testing.T) {
	for _, lang := range []string{"en", "ro"} {
		mail := generatedUserEmail{Subject: "S", Lang: lang, Name: "<b>x</b>", Username: "user1", Password: "pw<1>", Host: "https://kn.example", Branding: "KN"}
		msg, err := mail.message("a@b.c")
		if err != nil {
			t.Fatal(err)
		}
		all := msg.PlainContent + msg.HTMLContent
		if strings.Contains(all, "mail.") || strings.Contains(all, "/contests/") {
			t.Errorf("%s: untranslated string or contest link without a contest: %s", lang, all)
		}
		if !strings.Contains(msg.PlainContent, ": user1\n") || !strings.Contains(msg.PlainContent, ": pw<1>\n") {
			t.Errorf("%s: plain is missing the login details: %s", lang, msg.PlainContent)
		}
		if !strings.Contains(msg.HTMLContent, ">user1</code>") || !strings.Contains(msg.HTMLContent, ">pw&lt;1&gt;</code>") || strings.Contains(msg.HTMLContent, "<b>") {
			t.Errorf("%s: html is missing the login details or does not escape: %s", lang, msg.HTMLContent)
		}

		mail.Contest = &kilonova.Contest{ID: 7, Name: "A & B"}
		msg, _ = mail.message("a@b.c")
		if !strings.Contains(msg.PlainContent, "A & B\n") || !strings.Contains(msg.PlainContent, " https://kn.example/contests/7\n") {
			t.Errorf("%s: plain is missing the contest: %s", lang, msg.PlainContent)
		}
		if !strings.Contains(msg.HTMLContent, `<a href="https://kn.example/contests/7">A &amp; B</a>`) {
			t.Errorf("%s: html is missing the contest link: %s", lang, msg.HTMLContent)
		}
	}
}
