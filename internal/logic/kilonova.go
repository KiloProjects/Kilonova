package logic

import (
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/rclient"
)

// Version is the version of the platform
const Version = "Beta v0.6.0"

type Kilonova struct {
	DB      *db.DB
	DM      datamanager.Manager
	RClient *rclient.RClient
	Debug   bool
	email   *Email
}

func New(db *db.DB, dm datamanager.Manager, rclient *rclient.RClient, debug bool) (*Kilonova, error) {
	email, err := NewEmail()
	if err != nil {
		return nil, err
	}

	email.SendEmail("alexv@siluta.ro", "Test Subject", "Dacă ai primit asta, ești boss de boss")

	return &Kilonova{db, dm, rclient, debug, email}, nil
}
