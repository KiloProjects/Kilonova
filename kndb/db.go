// Package kndb provides a wrapper for the database
package kndb

import (
	"log"

	"github.com/KiloProjects/Kilonova/cache"
	"github.com/KiloProjects/Kilonova/internal/models"
	"gorm.io/gorm"
)

// DB is the wrapper for the DB
type DB struct {
	DB     *gorm.DB
	logger *log.Logger
	cache  *cache.Cache
}

// New returns a new DB instance with the specified gorm DB
func New(db *gorm.DB, logger *log.Logger) (*DB, error) {
	cache, err := cache.New()
	if err != nil {
		return nil, err
	}
	return &DB{db, logger, cache}, nil
}

// Save adds an arbitrary struct to the DB (or if the primary key is set, overwrites the value)
func (d *DB) Save(data interface{}) error {
	return d.DB.Save(data).Error
}

// AutoMigrate calls db.AutoMigrate for every struct in common/dbModels.go
func (d *DB) AutoMigrate() {
	d.DB.AutoMigrate(&models.User{})
	d.DB.AutoMigrate(&models.Problem{})
	d.DB.AutoMigrate(&models.Test{})
	d.DB.AutoMigrate(&models.EvalTest{})
	d.DB.AutoMigrate(&models.Task{})
}
