package judge

import (
	"fmt"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/grader/box"
	"gorm.io/gorm"
)

type taskStatusUpdate struct {
	id     uint
	status int
}

func (u taskStatusUpdate) Update(db *gorm.DB) error {
	fmt.Printf("TASK STATUS UPDATE: %d\n\n", u.status)
	return db.Model(&common.Task{}).Where("id = ?", u.id).Update("status", u.status).Error
}

type taskScoreUpdate struct {
	id    uint
	score int
}

func (u taskScoreUpdate) Update(db *gorm.DB) error {
	fmt.Printf("TASK SCORE UPDATE: %d\n\n", u.score)
	return db.Model(&common.Task{}).Where("id = ?", u.id).Update("score", u.score).Error
}

type testOutputUpdate struct {
	id     uint
	output string
	score  int
}

func (u testOutputUpdate) Update(db *gorm.DB) error {
	fmt.Printf("OUTPUT UPDATE: %s\nSCORE:%d\n\n", u.output, u.score)
	return db.Model(&common.EvalTest{}).Where("id = ?", u.id).
		Updates(map[string]interface{}{"score": u.score, "output": u.output, "done": true}).Error
}

type taskCompileUpdate struct {
	id             uint
	compileMessage string
	isFatal        bool
}

func (u taskCompileUpdate) Update(db *gorm.DB) error {
	fmt.Println("TASK COMPILE UPDATE:", u.compileMessage, u.isFatal)
	if u.compileMessage == "" {
		u.compileMessage = "No errors reported."
	}
	return db.Model(&common.Task{}).Where("id = ?", u.id).
		Updates(map[string]interface{}{"compile_error": u.isFatal, "compile_message": u.compileMessage}).Error
}

type testMetaUpdate struct {
	id   uint
	meta *box.MetaFile
}

func (u testMetaUpdate) Update(db *gorm.DB) error {
	t := common.EvalTest{}
	t.ID = u.id
	fmt.Printf("META UPDATE: %#v", u.meta)
	return db.Model(&t).Updates(map[string]interface{}{"time": u.meta.Time, "wall_time": u.meta.WallTime, "memory": u.meta.CgMem}).Error
}
