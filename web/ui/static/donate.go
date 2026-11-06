// Package static renders the mostly-static pages: donations, privacy policy and terms of service.
package static

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/web/ui"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

type DonateParams struct {
	Donations []*kilonova.Donation
	// Status is the ?status= of a Stripe redirect: "success", "cancelled" or empty.
	Status string

	// Donation method flags; an empty value hides that method.
	StripeButtonID    string
	StripePK          string
	StripePaymentLink string
	PaypalID          string
	BMACName          string
}

func Donate(ctx context.Context, p DonateParams) Node {
	authed := user.UserBriefContext(ctx)
	admin := authed.IsAdmin()
	return Group{
		If(p.Status == "success", Div(Class("segment-panel reset-list"), H1(ui.T(ctx, "donate_success")))),
		If(p.Status == "cancelled", Div(Class("segment-panel reset-list"), H1(ui.T(ctx, "donate_cancelled")))),
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "support_kn")),
			P(ui.T(ctx, "donate_par")),
		),
		Div(Class("segment-panel"),
			H1(ui.T(ctx, "donation_methods"), Text(":")),
			Ul(
				If(p.StripeButtonID != "", Group{
					Script(Async(), Src("https://js.stripe.com/v3/buy-button.js")),
					El("stripe-buy-button", ID("stripe"),
						Attr("buy-button-id", p.StripeButtonID),
						Attr("publishable-key", p.StripePK),
						Attr("__locale-override", util.LanguageContext(ctx)),
						Iff(authed != nil, func() Node {
							var email string
							if full := user.UserFullContext(ctx); full != nil {
								email = full.Email
							}
							return Group{
								Attr("client-reference-id", strconv.Itoa(authed.ID)),
								Attr("customer-email", email),
							}
						}),
					),
				}),
				Iff(p.StripePaymentLink != "", func() Node {
					href := p.StripePaymentLink + "?"
					if authed != nil {
						href += "client_reference_id=" + strconv.Itoa(authed.ID)
					}
					return Li(Class("mb-2"),
						Text("Stripe: "),
						A(Target("_blank"), Class("btn btn-blue"), Href(href), I(Class("fab fa-stripe mr-2")), Text(" Donate with Stripe")), Text(" "),
						ui.T(ctx, "donate_stripe_disclaimer"),
					)
				}),
				If(p.PaypalID != "", Li(Class("mb-2"),
					Text("PayPal: "),
					A(Target("_blank"), Class("btn btn-blue"), Href("https://www.paypal.com/donate/?hosted_button_id="+url.QueryEscape(p.PaypalID)), I(Class("fab fa-paypal mr-2")), Text(" Donate with PayPal")), Text(" "),
					ui.T(ctx, "donate_paypal_disclaimer"),
				)),
				If(p.BMACName != "", Li(Class("mb-2"),
					Text("Buy Me A Coffee: "),
					A(Target("_blank"), Href("https://www.buymeacoffee.com/"+url.PathEscape(p.BMACName)),
						Img(Class("inline"), Src("https://img.buymeacoffee.com/button-api/?text=Donate&emoji=&slug="+url.QueryEscape(p.BMACName)+"&button_colour=FFDD00&font_family=Lato&outline_colour=000000&coffee_colour=ffffff")),
					), Text(" "),
					ui.T(ctx, "donate_bmac_disclaimer"),
				)),
			),
		),
		If(len(p.Donations) > 0, Div(Class("segment-panel reset-list"),
			H1(ui.T(ctx, "past_donations")),
			Ul(Map(p.Donations, func(d *kilonova.Donation) Node { return donation(ctx, d, admin) })),
		)),
		If(admin, adminForms(ctx)),
	}
}

func donation(ctx context.Context, d *kilonova.Donation, admin bool) Node {
	var donor Node
	switch {
	case d.User != nil:
		donor = A(Href("/profile/"+d.User.Name), Text(d.User.Name), If(admin, Textf(" (#%d)", d.User.ID)))
	case d.RealName != "":
		donor = Text(d.RealName)
	default:
		donor = Text("???")
	}

	var amount Node
	if d.Type == kilonova.DonationTypeOneTime {
		amount = Textf("%.2f %s", d.Amount, d.Currency)
	} else {
		var period Node
		switch d.Type {
		case kilonova.DonationTypeMonthly:
			period = ui.T(ctx, "per_month")
		case kilonova.DonationTypeYearly:
			period = ui.T(ctx, "per_year")
		}
		amount = Group{
			El("kn-glossary", Attr("name", "lifetime_donation_amount"), Attr("content", fmt.Sprint(DonationSum(d, time.Now()))+" "+d.Currency)),
			Textf(" (%.2f %s", d.Amount, d.Currency), period, Text(")"),
		}
	}

	return Li(
		donor, Text(" - "), amount,
		Text(" ("), ui.ServerTimestamp(d.DonatedAt), Text(")"),
		If(admin, Ul(
			Li(ui.T(ctx, "donation_method"), Text(": "+string(d.Source))),
			Li(ui.T(ctx, "donation_type"), Text(": "+string(d.Type))),
			Li(ui.T(ctx, "transaction_id"), Text(": "+d.TransactionID)),
			Iff(d.CancelledAt != nil, func() Node {
				return Li(ui.T(ctx, "cancelled_at"), Text(": "), ui.ServerTimestamp(*d.CancelledAt))
			}),
			If(d.CancelledAt == nil && d.Type != kilonova.DonationTypeOneTime, Li(
				ui.Button(Attr("onclick", fmt.Sprintf("endSubscription(%d)", d.ID)), ui.T(ctx, "button.cancel")),
			)),
		)),
	)
}

// DonationSum is the total a donation has brought in by now (or by its cancellation).
func DonationSum(d *kilonova.Donation, now time.Time) float64 {
	endTime := now
	if d.CancelledAt != nil {
		endTime = *d.CancelledAt
	}
	numMonths := (endTime.Year()-d.DonatedAt.Year())*12 + int(endTime.Month()-d.DonatedAt.Month()) + 1
	if endTime.Day() < d.DonatedAt.Day() {
		numMonths--
	}
	switch d.Type {
	case kilonova.DonationTypeMonthly:
		return d.Amount * float64(numMonths)
	case kilonova.DonationTypeYearly:
		numMonths /= 12
		return d.Amount * float64(numMonths)
	default:
		return d.Amount
	}
}

func adminForms(ctx context.Context) Node {
	field := func(label Node, input Node) Node {
		return Label(Class("block my-2"), Span(Class("form-label"), label, Text(":")), Text(" "), input)
	}
	sel := func(key, id string, opts ...Node) Node {
		return Label(Class("block mb-2"),
			Span(Class("form-label"), ui.T(ctx, key), Text(":")), Text(" "),
			Select(ID(id), Class("form-select"), AutoComplete("off"), Group(opts)),
		)
	}
	return Group{
		Script(Raw(`
    async function endSubscription(id) {
        if(!(await bundled.confirm(bundled.getText("end_sub_confirmation")))) {
            return
        }
        let res = await bundled.postCall("/admin/endSubscription", {id})
        if(res.status === "error") {
            bundled.apiToast(res)
            return
        }
        window.location.reload()
    }
`)),
		Form(Class("segment-panel"), ID("addDonationForm"),
			H1(ui.T(ctx, "add_donation")),
			field(ui.T(ctx, "donation_time"), Input(Type("datetime-local"), ID("donation-time"), Class("form-input"), Required())),
			field(Group{ui.T(ctx, "username"), Text(" (empty = anonymous)")}, Input(Type("text"), ID("donation-username"), Class("form-input"))),
			field(ui.T(ctx, "amount"), Input(Type("number"), ID("donation-amount"), Class("form-input"), Step("0.01"), Min("0"), Required())),
			sel("currency", "donation-currency",
				Option(Value("EUR"), Selected(), Text("🇪🇺 EUR €")),
				Option(Value("USD"), Text("🇺🇸 USD $")),
				Option(Value("RON"), Text("🇷🇴 RON")),
				Option(Value("GBP"), Text("🇬🇧 GBP £")),
			),
			sel("donation_method", "donation-source",
				Option(Value("buymeacoffee"), Selected(), Text("Buy Me A Coffee")),
				Option(Value("paypal"), Text("PayPal")),
				Option(Value("stripe"), Text("Stripe")),
				Option(Value("other"), ui.T(ctx, "other")),
			),
			sel("donation_type", "donation-type",
				Option(Value("onetime"), Selected(), Text("One-Time")),
				Option(Value("monthly"), Text("Monthly")),
				Option(Value("yearly"), Text("Yearly")),
			),
			field(ui.T(ctx, "transaction_id"), Input(Type("text"), ID("donation-transactionid"), Class("form-input"), Required())),
			ui.SubmitButton(ui.T(ctx, "button.create")),
		),
		Script(Raw(`
    async function addDonation(e) {
        e.preventDefault();
        let args = {
            donated_at: bundled.dayjs(document.getElementById("donation-time").value).format(),
            amount: document.getElementById("donation-amount").value,
            currency: document.getElementById("donation-currency").value,
            username: document.getElementById("donation-username").value,

            source: document.getElementById("donation-source").value,
            type: document.getElementById("donation-type").value,
            transaction_id: document.getElementById("donation-transactionid").value,
        }
        const res = await bundled.postCall("/admin/addDonation", args)
        if(res.status === "success") {
            window.location.reload();
            return
        }
        bundled.apiToast(res)
    }
    document.getElementById("addDonationForm").addEventListener("submit", addDonation)
`)),
	}
}
