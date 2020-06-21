package judge

import (
	"fmt"

	"github.com/jinzhu/gorm"
)

type taskStatusUpdate struct {
	id     uint
	status int
}

func (u taskStatusUpdate) Update(db *gorm.DB) error {
	// no
	// fmt.Printf("RECEIVED UPDATE STATUS FOR TASK %d: STATUS %d\n", u.id, u.status)
	return nil
}

type testStatusUpdate struct {
	id     uint
	result bool
}

func (u testStatusUpdate) Update(db *gorm.DB) error {
	res := ""
	if u.result {
		res = "IT'S FUCKING GOOD, CA-CHING"
	} else {
		res = "(in trump voice) WROOOONG"
	}
	fmt.Printf("RECEIVED UPDATE STATUS FOR TEST %d: %s\n", u.id, res)
	return nil
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
	return nil
}
