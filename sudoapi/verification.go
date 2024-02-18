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
)

var verificationEmailTempl = template.Must(template.New("emailTempl").Parse(`Hey, {{.Name}}!

Încă nu ți-ai verificat emailul. Te rog să intri pe acest link ca să fim siguri că ești tu: {{.HostPrefix}}/verify/{{.VID}}

Dacă acest cont nu este al tău, poți ignora acest email.

------
Echipa Kilonova
https://kilonova.ro/`))

// SendVerificationEmail updates the user metadata with an unverified email status and sends an email with the hard-coded template to the desired user.
// Please provide a good context.
//
// NOTE: I think the user update breaks some single responsibility principle or something, but I think most places this could be used also does this, so meh.
//
// If `email` is different than the user's email, the email address is also updated.
func (s *BaseAPI) SendVerificationEmail(ctx context.Context, userID int, name, email string) *StatusError {
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
	if err := verificationEmailTempl.Execute(&b, struct {
		Name       string
		VID        string
		HostPrefix string
	}{
		Name:       name,
		VID:        vid,
		HostPrefix: config.Common.HostPrefix,
	}); err != nil {
		zap.S().Error("Error rendering verification email:", err)
		return Statusf(500, "Error rendering email")
	}
	if err := s.mailer.SendEmail(&kilonova.MailerMessage{Subject: "Verifică-ți adresa de mail", PlainContent: b.String(), To: email}); err != nil {
		zap.S().Warn(err)
		return Statusf(500, "Error sending email")
	}

	return nil
}

func (s *BaseAPI) CheckVerificationEmail(ctx context.Context, vid string) bool {
	// TODO: Why did i write context.Background here?
	val, err := s.db.GetVerification(context.Background(), vid)
	return err == nil && val > 0
}

func (s *BaseAPI) GetVerificationUser(ctx context.Context, vid string) (int, *StatusError) {
	id, err := s.db.GetVerification(ctx, vid)
	if err != nil || id == -1 {
		return -1, Statusf(404, "Verification code doesn't exist")
	}
	return id, nil
}

func (s *BaseAPI) ConfirmVerificationEmail(vid string, user *kilonova.UserBrief) *StatusError {
	if err := s.db.RemoveVerification(context.Background(), vid); err != nil {
		return Statusf(500, "Couldn't delete verification code.")
	}

	ttrue := true
	return s.updateUser(context.Background(), user.ID, kilonova.UserFullUpdate{VerifiedEmail: &ttrue})
}

func (s *BaseAPI) MailerEnabled() bool {
	return config.Email.Enabled && s.mailer != nil
}
