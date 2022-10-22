package sudoapi

import (
	"bytes"
	"context"
	"text/template"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var emailTemplate = template.Must(template.New("emailTempl").Parse(`Hey, {{.Name}}!

Încă nu ți-ai verificat e-mail-ul. Vă rugăm să intrați pe acest link  a fi siguri că ești tu: {{.HostPrefix}}/verify/{{.VID}}

Dacă acest cont nu este al tău, poți ignora acest e-mail.

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
	if s.mailer == nil {
		zap.S().Error("SendVerificationEmail called, but no mailer was provided to *sudoapi.BaseAPI")
		return Statusf(500, "Can't send email")
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
	if err := emailTemplate.Execute(&b, struct {
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
	if err := s.mailer.SendEmail(&kilonova.MailerMessage{Subject: "Verify your email address", PlainContent: b.String(), To: email}); err != nil {
		zap.S().Warn(err)
		return Statusf(500, "Error sending email")
	}

	return nil
}

func (s *BaseAPI) CheckVerificationEmail(ctx context.Context, vid string) bool {
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
