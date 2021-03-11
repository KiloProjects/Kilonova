package logic

import (
	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/rclient"
)

// Version is the version of the platform
const Version = "v0.7.0 \"Zinc\""

type Kilonova struct {
	DM      kilonova.DataStore
	RClient *rclient.RClient
	Debug   bool
	mailer  kilonova.Mailer

	userv kilonova.UserService
	tserv kilonova.TestService
}

func New(db kilonova.TypeServicer, dm kilonova.DataStore, rclient *rclient.RClient, debug bool) (*Kilonova, error) {
	mailer, err := NewMailer()
	if err != nil {
		return nil, err
	}

	return &Kilonova{dm, rclient, debug, mailer, db.UserService(), db.TestService()}, nil
}
