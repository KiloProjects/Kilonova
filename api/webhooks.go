package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"

	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

type bmacEvent struct {
	Type    string          `json:"type"`
	Created int             `json:"created"`
	Data    json.RawMessage `json:"data"`
}

type donationData struct {
	Amount        float64 `json:"amount"`
	Currency      string  `json:"currency"`
	TransactionID string  `json:"transaction_id"`

	Name  string `json:"supporter_name"`
	Email string `json:"supporter_email"`
	Note  string `json:"support_note"`
}

type membershipDonationData struct {
	donationData
	PSPID     string `json:"psp_id"`
	LevelName string `json:"membership_level_name"`
}

// bmacEvent handles an event from Buy Me A Coffee
func (s *API) bmacEvent(w http.ResponseWriter, r *http.Request) {
	if config.Donations.BMACWebhookSecret == "" {
		zap.S().Warn("bmac_event was POSTed but no signature was specified in config file")
		errorData(w, "BMAC secret not rolled out", 400)
	}

	if r.Header.Get("X-Signature-Sha256") == "" {
		errorData(w, "Invalid Signature", 400)
		return
	}
	mac := hmac.New(sha256.New, []byte(config.Donations.BMACWebhookSecret))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		zap.S().Warn(err)
		errorData(w, "Couldn't read body to buffer", 500)
		return
	}
	mac.Write(buf.Bytes())
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	if expectedMAC != r.Header.Get("X-Signature-Sha256") {
		errorData(w, "Invalid signature", 500)
		return
	}

	var data bmacEvent

	if err := json.NewDecoder(&buf).Decode(&data); err != nil {
		zap.S().Warn(err)
		errorData(w, "Invalid JSON", 400)
		return
	}

	switch data.Type {
	case "donation.created":
		var donation donationData
		if err := json.Unmarshal(data.Data, &donation); err != nil {
			zap.S().Warn(err)
			return
		}
		s.base.LogToDiscord(r.Context(), "Donation for %f %s created. You have to manually add it to donations page (name: %s, email: %s, message: %q) (transaction id: %s)", donation.Amount, donation.Currency, donation.Name, donation.Email, donation.Note, donation.TransactionID)
	case "membership.started":
		var membership membershipDonationData
		if err := json.Unmarshal(data.Data, &membership); err != nil {
			zap.S().Warn(err)
			return
		}
		s.base.LogToDiscord(r.Context(), "Membership (%f %s) created. You have to manually add it to donations page (level: %s, name: %s, email: %s, message: %q) (PSP id: %s)", membership.Amount, membership.Currency, membership.LevelName, membership.Name, membership.Email, membership.Note, membership.PSPID)
	case "membership.cancelled":
		var membership membershipDonationData
		if err := json.Unmarshal(data.Data, &membership); err != nil {
			zap.S().Warn(err)
			return
		}
		s.base.LogToDiscord(r.Context(), "Membership **cancelled**. You have to manually remove it from donations page (level: %s, name: %s, email: %s, message: %q) (PSP id: %s)", membership.LevelName, membership.Name, membership.Email, membership.Note, membership.PSPID)
	}

	returnData(w, "Logged event")
}
