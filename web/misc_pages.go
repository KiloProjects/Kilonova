package web

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/KiloProjects/kilonova/web/ui/proposer"
	"github.com/KiloProjects/kilonova/web/ui/resources"
	"github.com/KiloProjects/kilonova/web/ui/static"
	"github.com/gorilla/schema"
)

func (rt *Web) privacyPolicy(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "privacyPolicy"),
		Content: static.PrivacyPolicy(r.Context(), kilonova.HostURL().JoinPath("").String()),
	})
}

func (rt *Web) termsOfService(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "termsOfService"),
		Content: static.TermsOfService(r.Context()),
	})
}

func (rt *Web) proposerPanel(w http.ResponseWriter, r *http.Request) {
	rt.runLayout(w, r, &LayoutParams{
		Title:   kilonova.GetText(util.Language(r), "panel.proposer"),
		Content: proposer.Panel(r.Context()),
	})
}

func (rt *Web) donationPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		donations, err := rt.base.Donations(r.Context())
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't get donations", slog.Any("err", err))
			donations = []*kilonova.Donation{}
		}

		rt.runLayout(w, r, &LayoutParams{
			Title: kilonova.GetText(util.Language(r), "donations"),
			Content: static.Donate(r.Context(), static.DonateParams{
				Donations:         donations,
				Status:            r.FormValue("status"),
				StripeButtonID:    flags.StripeButtonID.Value(),
				StripePK:          flags.StripePK.Value(),
				StripePaymentLink: flags.StripePaymentLink.Value(),
				PaypalID:          flags.PaypalID.Value(),
				BMACName:          flags.BuyMeACoffeeName.Value(),
			}),
		})
	}
}

func (rt *Web) externalResources() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// The listing is still a TODO page, so nothing is loaded yet.
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "external_resources"),
			Content: resources.TODO(r.Context()),
		})
	}
}

//nolint:unused
func (rt *Web) externalResource() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runLayout(w, r, &LayoutParams{
			Title:   util.ExternalResource(r).Name + " | Kilonova",
			Content: resources.TODO(r.Context()),
		})
	}
}

func (rt *Web) createExternalResourceView() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt.runLayout(w, r, &LayoutParams{
			Title:   kilonova.GetText(util.Language(r), "header.external_resources.new"),
			Content: resources.NewForm(r.Context(), util.Problem(r).ID),
		})
	}
}

func (rt *Web) createExternalResource() http.HandlerFunc {
	decoder := schema.NewDecoder()
	decoder.IgnoreUnknownKeys(true)
	decoder.SetAliasTag("json")
	return func(w http.ResponseWriter, r *http.Request) {
		var params struct {
			Name        string                `json:"name"`
			Description string                `json:"description"`
			URL         string                `json:"url"`
			Type        kilonova.ResourceType `json:"type"`
			PreApproved bool                  `json:"preApproved"`
		}
		r.ParseForm()
		if err := decoder.Decode(&params, r.Form); err != nil {
			htmxErrorToast(w, r, "Invalid request body")
			rt.statusPage(w, r, 400, "Invalid request body")
			return
		}
		if params.Name == "" || params.URL == "" || params.Type == "" {
			htmxErrorToast(w, r, "Invalid request body")
			rt.statusPage(w, r, 400, "Invalid request body")
			return
		}
		if params.PreApproved {
			params.PreApproved = user.UserBrief(r).IsAdmin()
		}

		res, err := rt.base.CreateExternalResource(
			r.Context(),
			params.Name, params.Description,
			params.URL,
			params.Type,
			user.UserBrief(r),
			util.Problem(r),
			params.PreApproved,
		)
		if err != nil {
			slog.WarnContext(r.Context(), "Couldn't create external resource", slog.Any("err", err))
			htmxErrorToast(w, r, "Couldn't create external resource")
			rt.statusPage(w, r, 500, "Couldn't create external resource")
			return
		}
		w.Header().Add("X-Resource-ID", fmt.Sprintf("%d", res.ID))
		htmxSuccessToast(w, r, "External resource created")
		w.Header().Add("HX-Redirect", fmt.Sprintf("/problems/%d", util.Problem(r).ID))
		http.Redirect(w, r, fmt.Sprintf("/problems/%d", util.Problem(r).ID), http.StatusOK)
	}
}
