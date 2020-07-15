package kndb

import (
	"github.com/KiloProjects/Kilonova/common"
)

// GetTaskByID returns a task with the specified ID
func (d *DB) GetTaskByID(id uint) (*common.Task, error) {
	var task common.Task
	if err := d.DB.
		Preload("Problem").Preload("User").Preload("Tests").Preload("Tests.Test").
		First(&task, id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// GetAllTasks returns all tasks
// TODO: Pagination
func (d *DB) GetAllTasks() ([]common.Task, error) {
	var tasks []common.Task
	if err := d.DB.Preload("Problem").Preload("User").Order("id").Find(&tasks).Error; err != nil {
		return nil, err
	}
	return tasks, nil
}
