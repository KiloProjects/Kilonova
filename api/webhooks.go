package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
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
	if flags.BMACWebhookSecret.Value() == "" {
		slog.WarnContext(r.Context(), "bmac_event was POSTed but no secret was specified in config file")
		errorData(w, "BMAC secret not rolled out", 400)
		return
	}

	if r.Header.Get("X-Signature-Sha256") == "" {
		errorData(w, "Invalid Signature", 400)
		return
	}
	mac := hmac.New(sha256.New, []byte(flags.BMACWebhookSecret.Value()))
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r.Body); err != nil {
		slog.WarnContext(r.Context(), "Couldn't read body to buffer", slog.Any("err", err))
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
		slog.WarnContext(r.Context(), "Invalid JSON", slog.Any("err", err))
		errorData(w, "Invalid JSON", 400)
		return
	}

	switch data.Type {
	case "donation.created":
		var donation donationData
		if err := json.Unmarshal(data.Data, &donation); err != nil {
			slog.WarnContext(r.Context(), "Invalid JSON donation data", slog.Any("err", err))
			return
		}
		s.base.LogToDiscord(r.Context(), "New Buy Me a Coffee donation. You have to manually add it to donations page",
			slog.String("amount", fmt.Sprintf("%f %s", donation.Amount, donation.Currency)),
			slog.String("name", donation.Name), slog.String("email", donation.Email), slog.String("note", donation.Note),
			slog.String("transaction_id", donation.TransactionID),
		)
	case "membership.started":
		var membership membershipDonationData
		if err := json.Unmarshal(data.Data, &membership); err != nil {
			slog.WarnContext(r.Context(), "Invalid JSON membership start data", slog.Any("err", err))
			return
		}
		s.base.LogToDiscord(r.Context(), "New Buy Me a Coffee membership. You have to manually add it to donations page",
			slog.String("amount", fmt.Sprintf("%f %s", membership.Amount, membership.Currency)), slog.String("level", membership.LevelName),
			slog.String("name", membership.Name), slog.String("email", membership.Email), slog.String("note", membership.Note),
			slog.String("psp_id", membership.PSPID),
		)
	case "membership.cancelled":
		var membership membershipDonationData
		if err := json.Unmarshal(data.Data, &membership); err != nil {
			slog.WarnContext(r.Context(), "Invalid JSON membership cancel data", slog.Any("err", err))
			return
		}
		s.base.LogToDiscord(r.Context(), "Buy Me a Coffee membership **cancelled**. You have to manually remove it from donations page",
			slog.String("level", membership.LevelName),
			slog.String("name", membership.Name), slog.String("email", membership.Email), slog.String("note", membership.Note),
			slog.String("psp_id", membership.PSPID),
		)
	}

	returnData(w, "Logged event")
}
