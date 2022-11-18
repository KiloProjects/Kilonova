package sudoapi

import (
	"bytes"
	"context"
	"text/template"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"go.uber.org/zap"
)

var forgotPwdTempl = template.Must(template.New("emailTempl").Parse(`Hey, {{.Name}}!

Cineva a solicitat o cerere de resetare a parolei pentru contul tău. 
Dacă tu ai fost cel care a trimis-o, accesează linkul următor pentru a schimba parola: {{.HostPrefix}}/resetPassword/{{.VID}} 

Dacă solicitarea nu a fost trimisă de tine, poți ignora acest email.

------
Echipa Kilonova
https://kilonova.ro/`))

// SendPasswordResetEmail sends a password reset email to the user.
// Please provide a good context.
func (s *BaseAPI) SendPasswordResetEmail(ctx context.Context, userID int, name, email string) *StatusError {
	if s.mailer == nil {
		zap.S().Error("SendPasswordResetEmail called, but no mailer was provided to *sudoapi.BaseAPI")
		return Statusf(500, "Can't send email")
	}

	vid, err := s.db.CreatePwdResetRequest(ctx, userID)
	if err != nil {
		zap.S().Warn(err)
		return Statusf(500, "Couldn't create password request code")
	}

	var b bytes.Buffer
	if err := forgotPwdTempl.Execute(&b, struct {
		Name       string
		VID        string
		HostPrefix string
	}{
		Name:       name,
		VID:        vid,
		HostPrefix: config.Common.HostPrefix,
	}); err != nil {
		zap.S().Error("Error rendering password request email:", err)
		return Statusf(500, "Error rendering email")
	}
	if err := s.mailer.SendEmail(&kilonova.MailerMessage{Subject: "Recuperare parolă kilonova.ro", PlainContent: b.String(), To: email}); err != nil {
		zap.S().Warn(err)
		return Statusf(500, "Error sending email")
	}

	return nil
}

func (s *BaseAPI) CheckPasswordResetRequest(ctx context.Context, rid string) bool {
	val, err := s.db.GetPwdResetRequest(context.Background(), rid)
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
