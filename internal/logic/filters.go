package logic

import (
	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (kn *Kilonova) FilterCode(s *kilonova.Submission, user *kilonova.User, db kilonova.DB) {
	if !util.IsSubmissionVisible(s, user, db) {
		s.Code = ""
	}
}
