package logic

import (
	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (kn *Kilonova) FilterCode(s *kilonova.Submission, user *kilonova.User, sserv kilonova.SubmissionService) {
	if !util.IsSubmissionVisible(s, user, sserv) {
		s.Code = ""
	}
}
