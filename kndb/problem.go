package kndb

import (
	"github.com/KiloProjects/Kilonova/common"
)

// GetProblemByID returns a problem with the specified ID
func (d *DB) GetProblemByID(id uint) (*common.Problem, error) {
	var problem common.Problem
	if err := d.DB.Set("gorm:auto_preload", true).First(&problem, id).Error; err != nil {
		return nil, err
	}
	return &problem, nil
}

// GetAllProblems returns a slice with all problems
// TODO: Pagination
func (d *DB) GetAllProblems() ([]common.Problem, error) {
	var problems []common.Problem
	if err := d.DB.Preload("Tests").Find(&problems).Error; err != nil {
		return nil, err
	}
	return problems, nil
}

// ProblemExists returns a bool if a problem with the specified name exists
func (d *DB) ProblemExists(name string) bool {
	var cnt int
	if err := d.DB.Model(&common.Problem{}).Where("lower(name) = lower(?)", name).Count(&cnt).Error; err != nil {
		return false
	}
	if cnt > 0 {
		return true
	}
	return false
}

// UpdateProblemField updates the field of a problem with the specified ID
// Since there are a lot of fields to the problem, I won't write a function for each and every one
func (d *DB) UpdateProblemField(id uint, fieldName string, fieldValue interface{}) error {
	return d.DB.Preload("Tests").Model(&common.Problem{}).Where("id = ?", id).Update(fieldName, fieldValue).Error
}

/********************************************************************************
 * LIMITS
 ********************************************************************************/

// UpdateLimit takes a problem id and a map[string]interface{} (which has the fields that will be changed)
func (d *DB) UpdateLimit(pbid uint, limit map[string]interface{}) error {
	return d.DB.Preload("Limits").Model(&common.Problem{}).Where("id = ?", pbid).Update(limit).Error
}
