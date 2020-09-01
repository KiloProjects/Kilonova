package kndb

import "github.com/KiloProjects/Kilonova/common"

func (d *DB) GetTestByID(id uint) (*common.Test, error) {
	var test common.Test
	if err := d.DB.First(&test, id).Error; err != nil {
		return nil, err
	}
	return &test, nil
}

func (d *DB) GetTestByVisibleID(pbid uint, vid uint) (*common.Test, error) {
	var test common.Test
	if err := d.DB.Where("problem_id = ? AND visible_id = ?", pbid, vid).First(&test).Error; err != nil {
		return nil, err
	}
	return &test, nil
}
