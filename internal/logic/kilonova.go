package logic

import (
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/internal/db"
)

type Kilonova struct {
	DB    *db.DB
	DM    datamanager.Manager
	Debug bool
}

func New(db *db.DB, dm datamanager.Manager, debug bool) (*Kilonova, error) {
	return &Kilonova{db, dm, debug}, nil
}
