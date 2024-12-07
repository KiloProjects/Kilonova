package sudoapi

import (
	"bytes"
	"context"
	"text/template"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"

	_ "embed"
)

//go:embed emails/forgotPassword.txt
var passwordForgotEmailText string
var forgotPwdTempl = template.Must(template.New("emailTempl").Parse(passwordForgotEmailText))

// SendPasswordResetEmail sends a password reset email to the user.
// Please provide a good context.
func (s *BaseAPI) SendPasswordResetEmail(ctx context.Context, userID int, name, email, lang string) *StatusError {
	if s.mailer == nil || !s.MailerEnabled() {
		zap.S().Error("SendPasswordResetEmail called, but no mailer was provided to *sudoapi.BaseAPI")
		return Statusf(500, "Mailer system was disabled by admins.")
	}

	vid, err := s.db.CreatePwdResetRequest(ctx, userID)
	if err != nil {
		zap.S().Warn(err)
		return Statusf(500, "Couldn't create password request code")
	}

	var b bytes.Buffer
	if err := forgotPwdTempl.ExecuteTemplate(&b, lang, struct {
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
		zap.S().Error("Error rendering password request email:", err)
		return Statusf(500, "Error rendering email")
	}
	if err := s.SendMail(ctx, &kilonova.MailerMessage{
		Subject:      kilonova.GetText(lang, "mail.subject.password_recovery"),
		PlainContent: b.String(),
		To:           email,
	}); err != nil {
		zap.S().Warn(err)
		return err
	}

	return nil
}

func (s *BaseAPI) CheckPasswordResetRequest(ctx context.Context, rid string) bool {
	val, err := s.db.GetPwdResetRequest(ctx, rid)
	return err == nil && val > 0
}

func (s *BaseAPI) GetPwdResetRequestUser(ctx context.Context, rid string) (int, *StatusError) {
	id, err := s.db.GetPwdResetRequest(ctx, rid)
	if err != nil || id == -1 {
		return -1, Statusf(404, "PwdResetRequest code doesn't exist")
	}
	return id, nil
}

func (s *BaseAPI) FinalizePasswordReset(ctx context.Context, rid string, newPassword string) *StatusError {
	userID, err := s.GetPwdResetRequestUser(ctx, rid)
	if err != nil {
		return err
	}

	if err := s.UpdateUserPassword(ctx, userID, newPassword); err != nil {
		return err
	}
	if err := s.db.RemovePwdResetRequest(ctx, rid); err != nil {
		zap.S().Warn(err)
		return Statusf(400, "Couldn't remove password reset request")
	}
	return nil
}
