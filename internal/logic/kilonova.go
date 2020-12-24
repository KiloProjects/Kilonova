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
}

func New(db *db.DB, dm datamanager.Manager, rclient *rclient.RClient, debug bool) (*Kilonova, error) {
	return &Kilonova{db, dm, rclient, debug}, nil
}
