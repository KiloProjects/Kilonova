package judge

import (
	"fmt"

	"github.com/KiloProjects/Kilonova/models"
	"github.com/jinzhu/gorm"
)

type taskStatusUpdate struct {
	id     uint
	status int
}

func (u taskStatusUpdate) Update(db *gorm.DB) error {
	fmt.Println("Updating", u.id, "with status", u.status)
	return db.Model(&models.Task{}).Where("id = ?", u.id).Update("status", u.status).Error
}

type taskScoreUpdate struct {
	id    uint
	score int
}

func (u taskScoreUpdate) Update(db *gorm.DB) error {
	return db.Model(&models.Task{}).Where("id = ?", u.id).Update("score", u.score).Error
}

type testOutputUpdate struct {
	id     uint
	output string
	score  int
}

func (u testOutputUpdate) Update(db *gorm.DB) error {
	fmt.Printf("RECEIVED UPDATE OUTPUT FOR TEST %d (given score %d): %s\n", u.id, u.score, u.output)
	return nil
}

type taskCompileUpdate struct {
	id             uint
	compileMessage string
	isFatal        bool
}

func (u taskCompileUpdate) Update(db *gorm.DB) error {
	if u.compileMessage == "" {
		u.compileMessage = "<empty>"
	}
	fmt.Printf("RECEIVED UPDATE COMPILE FOR TASK %d (is fatal: %t): %s\n", u.id, u.isFatal, u.compileMessage)
	db.Model(&models.Task{}).Where("id = ?", u.id).Update("compile_error", u.isFatal)
	db.Model(&models.Task{}).Where("id = ?", u.id).Update("compile_message", u.compileMessage)
	return nil
}
