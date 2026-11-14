package sudoapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
)

// SendPasswordResetEmail sends a password reset email to the user.
// Please provide a good context.
func (s *BaseAPI) SendPasswordResetEmail(ctx context.Context, userID int, name, email, lang string) error {
	if s.mailer == nil || !s.MailerEnabled() {
		slog.ErrorContext(ctx, "SendPasswordResetEmail called, but no mailer was provided to *sudoapi.BaseAPI")
		return errors.New("mailer system was disabled by admins")
	}

	vid, err := s.db.CreatePwdResetRequest(ctx, userID)
	if err != nil {
		slog.WarnContext(ctx, "Could not create password reset request", slog.Any("err", err))
		return errors.New("couldn't create password request code")
	}

	msg, err := linkEmail{
		Kind:     "password_recovery",
		Lang:     lang,
		Name:     name,
		URL:      s.hostURL.JoinPath("resetPassword", vid).String(),
		Host:     s.hostURL.String(),
		Branding: flags.EmailBranding.Of(s.flagset),
	}.message(email)
	if err != nil {
		slog.ErrorContext(ctx, "Error rendering password request email", slog.Any("err", err))
		return fmt.Errorf("error rendering email")
	}
	if err := s.SendMail(ctx, msg); err != nil {
		slog.WarnContext(ctx, "Error sending email", slog.Any("err", err), slog.String("kind", "password_recovery"))
		return err
	}

	return nil
}

func (s *BaseAPI) CheckPasswordResetRequest(ctx context.Context, rid string) bool {
	val, err := s.db.GetPwdResetRequest(ctx, rid)
	return err == nil && val > 0
}

func (s *BaseAPI) GetPwdResetRequestUser(ctx context.Context, rid string) (int, error) {
	id, err := s.db.GetPwdResetRequest(ctx, rid)
	if err != nil || id == -1 {
		return -1, Statusf(404, "PwdResetRequest code doesn't exist")
	}
	return id, nil
}

func (s *BaseAPI) FinalizePasswordReset(ctx context.Context, rid string, newPassword string) error {
	userID, err := s.GetPwdResetRequestUser(ctx, rid)
	if err != nil {
		return err
	}

	if err := s.UpdateUserPassword(ctx, userID, newPassword); err != nil {
		return err
	}
	if err := s.db.RemovePwdResetRequest(ctx, rid); err != nil {
		slog.WarnContext(ctx, "Couldn't remove password reset request", slog.Any("err", err))
		return Statusf(400, "Couldn't remove password reset request")
	}
	return nil
}
