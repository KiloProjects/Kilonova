package logic

import (
	"github.com/KiloProjects/kilonova"
)

type Kilonova struct {
	DM     kilonova.DataStore
	Debug  bool
	mailer kilonova.Mailer

	DB kilonova.DB
}

func New(db kilonova.DB, dm kilonova.DataStore, debug bool) (*Kilonova, error) {
	mailer, err := NewMailer()
	if err != nil {
		return nil, err
	}

	return &Kilonova{dm, debug, mailer, db}, nil
}
