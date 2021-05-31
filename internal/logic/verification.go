package logic

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
)

var emailTempl = `Hey, {{.Name}}!

Încă nu ți-ai verificat e-mail-ul. Vă rugăm să intrați pe acest link  a fi siguri că ești tu: {{.HostPrefix}}/verify/{{.VID}}

Dacă acest cont nu este al tău, poți ignora acest e-mail.

------
Echipa Kilonova
https://kilonova.ro/`
var emailT = template.Must(template.New("emailTempl").Parse(emailTempl))

// CreateVerification creates a new verification request
func (kn *Kilonova) CreateVerification(userid int) (string, error) {
	return kn.DB.CreateVerification(context.Background(), userid)
}

func (kn *Kilonova) SendVerificationEmail(email, name string, uid int) error {
	type emTp struct {
		Name       string
		Email      string
		VID        string
		HostPrefix string
	}
	vid, err := kn.CreateVerification(uid)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	if err := emailT.Execute(&b, emTp{Name: name, Email: email, VID: vid, HostPrefix: config.Common.HostPrefix}); err != nil {
		log.Fatal("Error rendering verification email:", err)
	}
	return kn.mailer.SendEmail(&kilonova.MailerMessage{Subject: "Verify your email address", PlainContent: b.String(), To: email})
}

func (kn *Kilonova) CheckVerificationEmail(vid string) bool {
	val, err := kn.DB.GetVerification(context.Background(), vid)
	if err != nil || val == 0 {
		return false
	}
	return true
}

var True = true

func (kn *Kilonova) ConfirmVerificationEmail(vid string, user *kilonova.User) error {
	if err := kn.DB.RemoveVerification(context.Background(), vid); err != nil {
		return err
	}

	return kn.DB.UpdateUser(context.Background(), user.ID, kilonova.UserUpdate{VerifiedEmail: &True})
}
