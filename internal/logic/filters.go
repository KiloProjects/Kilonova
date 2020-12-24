package logic

import (
	"github.com/KiloProjects/Kilonova/internal/db"
	"github.com/KiloProjects/Kilonova/internal/util"
)

func (kn *Kilonova) FilterCode(s *db.Submission, user *db.User) {
	if !util.IsSubmissionVisible(s, user) {
		s.Code = ""
	}
}
