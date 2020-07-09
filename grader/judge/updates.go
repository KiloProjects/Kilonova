package judge

import (
	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/grader/box"
	"github.com/jinzhu/gorm"
)

type taskStatusUpdate struct {
	id     uint
	status int
}

func (u taskStatusUpdate) Update(db *gorm.DB) error {
	return db.Model(&common.Task{}).Where("id = ?", u.id).Update("status", u.status).Error
}

type taskScoreUpdate struct {
	id    uint
	score int
}

func (u taskScoreUpdate) Update(db *gorm.DB) error {
	return db.Model(&common.Task{}).Where("id = ?", u.id).Update(map[string]interface{}{"score": u.score}).Error
}

type testOutputUpdate struct {
	id     uint
	output string
	score  int
}

func (u testOutputUpdate) Update(db *gorm.DB) error {
	return db.Model(&common.EvalTest{}).Where("id = ?", u.id).
		Update(map[string]interface{}{"score": u.score, "output": u.output, "done": true}).Error
}

type taskCompileUpdate struct {
	id             uint
	compileMessage string
	isFatal        bool
}

func (u taskCompileUpdate) Update(db *gorm.DB) error {
	if u.compileMessage == "" {
		u.compileMessage = "No errors reported."
	}
	return db.Model(&common.Task{}).Where("id = ?", u.id).
		Update(map[string]interface{}{"compile_error": u.isFatal, "compile_message": u.compileMessage}).Error
}

type testMetaUpdate struct {
	id   uint
	meta *box.MetaFile
}

func (u testMetaUpdate) Update(db *gorm.DB) error {
	t := common.EvalTest{}
	t.ID = u.id
	return db.Model(&t).Update(map[string]interface{}{"time": u.meta.Time, "wall_time": u.meta.WallTime, "memory": u.meta.CgMem}).Error
}
