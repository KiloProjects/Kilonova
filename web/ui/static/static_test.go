package static

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	. "maragu.dev/gomponents"
)

var base = context.WithValue(context.Background(), util.LangKey, "en")

func as(u *kilonova.UserBrief) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

func TestDonate(t *testing.T) {
	dons := []*kilonova.Donation{
		{ID: 3, User: &kilonova.UserBrief{ID: 9, Name: "<bob>"}, Amount: 5, Currency: "EUR", Type: kilonova.DonationTypeMonthly, TransactionID: "tx1"},
		{ID: 4, RealName: "Anon", Amount: 2.5, Currency: "RON", Type: kilonova.DonationTypeOneTime},
	}
	t.Run("visitors see donors but no admin tools", func(t *testing.T) {
		got := render(t, Donate(base, DonateParams{Donations: dons}))
		must(t, got, `href="/profile/&lt;bob&gt;"`, "2.50 RON", "kn-glossary", "(5.00 EUR/ month)")
		mustNot(t, got, "(#9)", "addDonationForm", "tx1", "endSubscription", "<li class=\"mb-2\">")
	})
	t.Run("admins see ids, details and cancel buttons", func(t *testing.T) {
		got := render(t, Donate(as(&kilonova.UserBrief{ID: 1, Admin: true}), DonateParams{Donations: dons}))
		must(t, got, "(#9)", "addDonationForm", "tx1", `onclick="endSubscription(3)"`)
		mustNot(t, got, "endSubscription(4)")
	})
	t.Run("methods appear per flag, with the user's reference", func(t *testing.T) {
		got := render(t, Donate(as(&kilonova.UserBrief{ID: 7}), DonateParams{StripePaymentLink: "https://buy.stripe.com/x", BMACName: "kn", Status: "success"}))
		must(t, got, `href="https://buy.stripe.com/x?client_reference_id=7"`, "buymeacoffee.com/kn", "Thank")
		mustNot(t, got, "paypal.com", "stripe-buy-button")
	})
	t.Run("stripe button carries customer reference only when authed", func(t *testing.T) {
		mustNot(t, render(t, Donate(base, DonateParams{StripeButtonID: "b"})), "client-reference-id")
		must(t, render(t, Donate(as(&kilonova.UserBrief{ID: 7}), DonateParams{StripeButtonID: "b"})), `client-reference-id="7"`, `buy-button-id="b"`)
	})
}

func TestDonationSum(t *testing.T) {
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	d := &kilonova.Donation{DonatedAt: start, Amount: 2, Type: kilonova.DonationTypeMonthly}
	if got := DonationSum(d, time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)); got != 4 {
		t.Fatalf("monthly sum = %v, want 4", got)
	}
	d.Type = kilonova.DonationTypeYearly
	if got := DonationSum(d, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)); got != 2 {
		t.Fatalf("yearly sum = %v, want 2", got)
	}
}

func TestLegal(t *testing.T) {
	must(t, render(t, PrivacyPolicy(base, "https://kn.test/")), `<a href="https://kn.test/">https://kn.test/</a>`, "Kilonova Privacy Policy")
	must(t, render(t, TermsOfService(base)), "Kilonova Terms of Service", `href="/privacyPolicy"`)
}

func render(t *testing.T, n Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatal(err)
	}
	return b.String()
}

func must(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Fatalf("missing %q in %s", w, got)
		}
	}
}

func mustNot(t *testing.T, got string, bad ...string) {
	t.Helper()
	for _, b := range bad {
		if strings.Contains(got, b) {
			t.Fatalf("unexpected %q in %s", b, got)
		}
	}
}
