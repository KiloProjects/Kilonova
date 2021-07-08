package kilonova

import (
	"bytes"
	"context"
	"log"
	"text/template"

	"github.com/KiloProjects/kilonova/internal/config"
)

var emailTempl = `Hey, {{.Name}}!

Încă nu ți-ai verificat e-mail-ul. Vă rugăm să intrați pe acest link  a fi siguri că ești tu: {{.HostPrefix}}/verify/{{.VID}}

Dacă acest cont nu este al tău, poți ignora acest e-mail.

------
Echipa Kilonova
https://kilonova.ro/`
var emailT = template.Must(template.New("emailTempl").Parse(emailTempl))

// user is used for the email, name and id fields
func SendVerificationEmail(user *User, db DB, mailer Mailer) error {
	type emTp struct {
		Name       string
		Email      string
		VID        string
		HostPrefix string
	}
	vid, err := db.CreateVerification(context.Background(), user.ID)
	if err != nil {
		return err
	}

	var b bytes.Buffer
	if err := emailT.Execute(&b, emTp{Name: user.Name, Email: user.Email, VID: vid, HostPrefix: config.Common.HostPrefix}); err != nil {
		log.Fatal("Error rendering verification email:", err)
	}
	return mailer.SendEmail(&MailerMessage{Subject: "Verify your email address", PlainContent: b.String(), To: user.Email})
}

func CheckVerificationEmail(db DB, vid string) bool {
	val, err := db.GetVerification(context.Background(), vid)
	if err != nil || val == 0 {
		return false
	}
	return true
}

var True = true

func ConfirmVerificationEmail(db DB, vid string, user *User) error {
	if err := db.RemoveVerification(context.Background(), vid); err != nil {
		return err
	}

	return db.UpdateUser(context.Background(), user.ID, UserUpdate{VerifiedEmail: &True})
}
