package kndb

import "github.com/KiloProjects/Kilonova/internal/models"

func (d *DB) GetTestByID(id uint) (*models.Test, error) {
	var test models.Test
	if err := d.DB.First(&test, id).Error; err != nil {
		return nil, err
	}
	return &test, nil
}

func (d *DB) GetTestByVisibleID(pbid, vid uint) (*models.Test, error) {
	var test models.Test
	if err := d.DB.Where("problem_id = ? AND visible_id = ?", pbid, vid).First(&test).Error; err != nil {
		return nil, err
	}
	return &test, nil
}

func (d *DB) UpdateVisibleID(testID, vid uint) error {
	return d.DB.Model(&models.Test{}).Where("id = ?", testID).Update("visible_id", vid).Error
}

func (d *DB) UpdateProblemTestVisibleID(pID, oldVID, vid uint) error {
	return d.DB.Model(&models.Test{}).Where("problem_id = ? AND visible_id = ?", pID, oldVID).Update("visible_id", vid).Error
}

func (d *DB) UpdateProblemTestScore(pID, oldVID uint, score int) error {
	return d.DB.Model(&models.Test{}).Where("problem_id = ? AND visible_id = ?", pID, oldVID).Update("score", score).Error
}
