// Package kndb provides a wrapper for the database
package kndb

import (
	"github.com/KiloProjects/Kilonova/common"
	"gorm.io/gorm"
)

// DB is the wrapper for the DB
type DB struct {
	DB *gorm.DB
}

// New returns a new DB instance with the specified gorm DB
func New(db *gorm.DB) *DB {
	return &DB{
		DB: db,
	}
}

// Save adds an arbitrary struct to the DB (or if the primary key is set, overwrites the value)
func (d *DB) Save(data interface{}) error {
	return d.DB.Save(data).Error
}

// AutoMigrate calls db.AutoMigrate for every struct in common/dbModels.go
func (d *DB) AutoMigrate() {
	d.DB.AutoMigrate(&common.User{})
	d.DB.AutoMigrate(&common.Problem{})
	d.DB.AutoMigrate(&common.Test{})
	d.DB.AutoMigrate(&common.EvalTest{})
	d.DB.AutoMigrate(&common.Task{})
}
