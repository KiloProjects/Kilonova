package kndb

import (
	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/proto"
	"gorm.io/gorm"
)

// GetSubmissionByID returns a submissionwith the specified ID
func (d *DB) GetSubmissionByID(id uint) (*models.Submission, error) {
	var sub models.Submission
	if err := d.DB.
		Preload("Problem").Preload("User").Preload("Tests", func(db *gorm.DB) *gorm.DB {
		return db.Order("eval_tests.id")
	}).Preload("Tests.Test").First(&sub, id).Error; err != nil {
		return nil, err
	}
	return &sub, nil
}

func (d *DB) UserSubmissionsOnProblem(userid uint, problemid uint) ([]models.Submission, error) {
	var submissions []models.Submission
	if err := d.DB.Preload("Problem").Preload("User").Preload("Tests", func(db *gorm.DB) *gorm.DB {
		return db.Order("eval_tests.id")
	}).Preload("Tests.Test").Where("user_id = ? and problem_id = ?", userid, problemid).
		Order("id desc").Find(&submissions).Error; err != nil {
		return nil, err
	}
	return submissions, nil
}

func (d *DB) MaxScoreFor(userid uint, problemid uint) (int, error) {
	subs, err := d.UserSubmissionsOnProblem(userid, problemid)
	if err != nil {
		return -1, err
	}
	maxscore := -1
	for _, subs := range subs {
		if subs.Score > maxscore {
			maxscore = subs.Score
		}
	}
	return maxscore, nil
}

// GetAllSubmissions returns all submissions
// TODO: Pagination
func (d *DB) GetAllSubmissions() ([]models.Submission, error) {
	var submissions []models.Submission
	if err := d.DB.Preload("Problem").Preload("User").Order("id desc").Find(&submissions).Error; err != nil {
		return nil, err
	}
	return submissions, nil
}

// For use by the judge

func (d *DB) GetWaitingSubmissions() ([]models.Submission, error) {
	var submissions []models.Submission
	err := d.DB.Where("status = ?", models.StatusWaiting).
		Preload("Tests").Preload("Problem").Preload("Tests.Test").
		Find(&submissions).Error

	if len(submissions) == 0 {
		return nil, err
	}
	return submissions, err
}

func (d *DB) UpdateSubmissionVisibility(id uint, visible bool) error {
	var tmp models.Submission
	tmp.ID = id
	return d.DB.Model(&tmp).Update("visible", visible).Error
}

func (d *DB) UpdateCompilation(c proto.CResponse) error {
	var tmp models.Submission
	tmp.ID = uint(c.ID)
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"compile_error":   !c.Success,
		"compile_message": c.Output,
	}).Error
}

func (d *DB) UpdateStatus(id uint, status, score int) error {
	var tmp models.Submission
	tmp.ID = id
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"status": status,
		"score":  score,
	}).Error
}

func (d *DB) UpdateEvalTest(r proto.TResponse, score int) error {
	var tmp models.EvalTest
	tmp.ID = uint(r.TID)
	return d.DB.Model(&tmp).Updates(map[string]interface{}{
		"output": r.Comments,
		"time":   r.Time,
		"memory": r.Memory,
		"score":  score,
		"done":   true,
	}).Error
}
