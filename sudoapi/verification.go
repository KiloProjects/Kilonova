package sudoapi

import (
	"bytes"
	"context"
	"errors"
	"text/template"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"

	_ "embed"
)

//go:embed emails/emailVerification.txt
var verificationEmailText string
var verificationEmailTempl = template.Must(template.New("emailTempl").Parse(verificationEmailText))

// SendVerificationEmail updates the user metadata with an unverified email status and sends an email with the hard-coded template to the desired user.
// Please provide a good context.
//
// NOTE: I think the user update breaks some single responsibility principle or something, but I think most places this could be used also does this, so meh.
//
// If `email` is different than the user's email, the email address is also updated.
func (s *BaseAPI) SendVerificationEmail(ctx context.Context, userID int, name, email, lang string) error {
	if s.mailer == nil || !s.MailerEnabled() || userID == 1 {
		zap.S().Infof("Auto confirming email for user #%d as valid", userID)

		t := true
		now := time.Now()
		if err := s.updateUser(ctx, userID, kilonova.UserFullUpdate{
			Email:            &email,
			VerifiedEmail:    &t,
			EmailVerifSentAt: &now,
		}); err != nil {
			return err
		}

		return nil // Statusf(500, "Mailer system was disabled by admins")
	}

	if user, err := s.UserFullByEmail(ctx, email); err != nil && !errors.Is(err, ErrNotFound) {
		zap.S().Warn(err)
		return Statusf(500, "Couldn't check if email is already used. Report to admin")
	} else if user != nil && user.ID != userID {
		return Statusf(400, "Email is already in use")
	}

	f := false
	now := time.Now()
	if err := s.updateUser(ctx, userID, kilonova.UserFullUpdate{
		Email:            &email,
		VerifiedEmail:    &f,
		EmailVerifSentAt: &now,
	}); err != nil {
		return err
	}

	vid, err := s.db.CreateVerification(ctx, userID)
	if err != nil {
		return Statusf(500, "Couldn't create verification code")
	}

	var b bytes.Buffer
	if err := verificationEmailTempl.ExecuteTemplate(&b, lang, struct {
		Name       string
		VID        string
		HostPrefix string
		Branding   string
	}{
		Name:       name,
		VID:        vid,
		HostPrefix: config.Common.HostPrefix,
		Branding:   EmailBranding.Value(),
	}); err != nil {
		zap.S().Error("Error rendering verification email:", err)
		return Statusf(500, "Error rendering email")
	}
	if err := s.SendMail(ctx, &kilonova.MailerMessage{
		Subject:      kilonova.GetText(lang, "mail.subject.verification"),
		PlainContent: b.String(),
		To:           email,
	}); err != nil {
		return err
	}

	return nil
}

func (s *BaseAPI) CheckVerificationEmail(ctx context.Context, vid string) bool {
	val, err := s.db.GetVerification(ctx, vid)
	return err == nil && val > 0
}

func (s *BaseAPI) GetVerificationUser(ctx context.Context, vid string) (int, error) {
	id, err := s.db.GetVerification(ctx, vid)
	if err != nil || id == -1 {
		return -1, Statusf(404, "Verification code doesn't exist")
	}
	return id, nil
}

func (s *BaseAPI) ConfirmVerificationEmail(ctx context.Context, vid string, user *kilonova.UserBrief) error {
	if err := s.db.RemoveVerification(ctx, vid); err != nil {
		return Statusf(500, "Couldn't delete verification code.")
	}

	ttrue := true
	return s.updateUser(ctx, user.ID, kilonova.UserFullUpdate{VerifiedEmail: &ttrue})
}

func (s *BaseAPI) MailerEnabled() bool {
	return config.Email.Enabled && s.mailer != nil
}
