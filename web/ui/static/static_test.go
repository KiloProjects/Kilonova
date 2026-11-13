package static

import (
	"context"
	"testing"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui/uitest"
	"github.com/stretchr/testify/require"
)

var base = context.WithValue(context.Background(), util.LangKey, "en")

func as(u *kilonova.UserBrief) context.Context { return context.WithValue(base, user.AuthedUserKey, u) }

func TestDonate(t *testing.T) {
	dons := []*kilonova.Donation{
		{ID: 3, User: &kilonova.UserBrief{ID: 9, Name: "<bob>"}, Amount: 5, Currency: "EUR", Type: kilonova.DonationTypeMonthly, TransactionID: "tx1"},
		{ID: 4, RealName: "Anon", Amount: 2.5, Currency: "RON", Type: kilonova.DonationTypeOneTime},
	}
	t.Run("visitors see donors but no admin tools", func(t *testing.T) {
		got := uitest.Render(t, Donate(base, DonateParams{Donations: dons}))
		uitest.Contains(t, got, `href="/profile/&lt;bob&gt;"`, "2.50 RON", "kn-glossary", "(5.00 EUR/ month)")
		uitest.NotContains(t, got, "(#9)", "addDonationForm", "tx1", "endSubscription", "<li class=\"mb-2\">")
	})
	t.Run("admins see ids, details and cancel buttons", func(t *testing.T) {
		got := uitest.Render(t, Donate(as(&kilonova.UserBrief{ID: 1, Admin: true}), DonateParams{Donations: dons}))
		uitest.Contains(t, got, "(#9)", "addDonationForm", "tx1", `onclick="endSubscription(3)"`)
		uitest.NotContains(t, got, "endSubscription(4)")
	})
	t.Run("methods appear per flag, with the user's reference", func(t *testing.T) {
		got := uitest.Render(t, Donate(as(&kilonova.UserBrief{ID: 7}), DonateParams{StripePaymentLink: "https://buy.stripe.com/x", BMACName: "kn", Status: "success"}))
		uitest.Contains(t, got, `href="https://buy.stripe.com/x?client_reference_id=7"`, "buymeacoffee.com/kn", "Thank")
		uitest.NotContains(t, got, "paypal.com", "stripe-buy-button")
	})
	t.Run("stripe button carries customer reference only when authed", func(t *testing.T) {
		uitest.NotContains(t, uitest.Render(t, Donate(base, DonateParams{StripeButtonID: "b"})), "client-reference-id")
		uitest.Contains(t, uitest.Render(t, Donate(as(&kilonova.UserBrief{ID: 7}), DonateParams{StripeButtonID: "b"})), `client-reference-id="7"`, `buy-button-id="b"`)
	})
}

func TestDonationSum(t *testing.T) {
	start := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	d := &kilonova.Donation{DonatedAt: start, Amount: 2, Type: kilonova.DonationTypeMonthly}
	require.Equal(t, 4.0, DonationSum(d, time.Date(2024, 3, 14, 0, 0, 0, 0, time.UTC)), "monthly sum")
	d.Type = kilonova.DonationTypeYearly
	require.Equal(t, 2.0, DonationSum(d, time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)), "yearly sum")
}

func TestLegal(t *testing.T) {
	uitest.Contains(t, uitest.Render(t, PrivacyPolicy(base, "https://kn.test/")), `<a href="https://kn.test/">https://kn.test/</a>`, "Kilonova Privacy Policy")
	uitest.Contains(t, uitest.Render(t, TermsOfService(base)), "Kilonova Terms of Service", `href="/privacyPolicy"`)
}
