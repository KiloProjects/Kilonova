package logic

import (
	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/rclient"
)

// Version is the version of the platform
const Version = "v0.7.0 \"Zinc\""

type Kilonova struct {
	DM     kilonova.DataStore
	Debug  bool
	mailer kilonova.Mailer

	userv kilonova.UserService
	tserv kilonova.TestService

	Sess  kilonova.Sessioner
	Verif kilonova.Verificationer
}

func New(db kilonova.TypeServicer, dm kilonova.DataStore, rclient *rclient.RClient, debug bool) (*Kilonova, error) {
	mailer, err := NewMailer()
	if err != nil {
		return nil, err
	}

	return &Kilonova{dm, debug, mailer, db.UserService(), db.TestService(), rclient, rclient}, nil
}
