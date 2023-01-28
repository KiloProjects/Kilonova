package api

import (
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
)

func (s *API) createContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name string `json:"name"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	id, err := s.base.CreateContest(r.Context(), args.Name, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

func (s *API) updateContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name *string `json:"name"`

		PublicJoin *bool `json:"public_join"`
		Visible    *bool `json:"visible"`

		StartTime *string `json:"start_time"`
		EndTime   *string `json:"end_time"`

		MaxSubs *int `json:"max_subs"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	var startTime, endTime *time.Time
	if args.StartTime != nil {
		t, err := time.Parse(time.RFC1123Z, *args.StartTime)
		if err != nil {
			errorData(w, "Invalid timestamp", 400)
			return
		}
		startTime = &t
	}
	if args.EndTime != nil {
		t, err := time.Parse(time.RFC1123Z, *args.EndTime)
		if err != nil {
			errorData(w, "Invalid timestamp", 400)
			return
		}
		endTime = &t
	}

	if err := s.base.UpdateContest(r.Context(), util.Contest(r).ID, kilonova.ContestUpdate{
		Name:       args.Name,
		PublicJoin: args.PublicJoin,
		Visible:    args.Visible,
		StartTime:  startTime,
		EndTime:    endTime,
		MaxSubs:    args.MaxSubs,
	}); err != nil {
		errorData(w, err, 500)
		return
	}

	returnData(w, "Contest updated")
}

func (s *API) updateContestProblems(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		List []int `json:"list"`
	}
	if err := parseJsonBody(r, &args); err != nil {
		err.WriteError(w)
		return
	}

	if args.List == nil {
		errorData(w, "You must specify a list of problems", 400)
	}

	list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.UpdateContestProblems(r.Context(), util.Contest(r).ID, list); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Updated contest problems")
}

func (s *API) getContest(w http.ResponseWriter, r *http.Request) {
	returnData(w, util.Contest(r))
}

func (s *API) addContestEditor(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"username"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.AddContestEditor(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Added contest editor")
}

func (s *API) addContestTester(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"username"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		err.WriteError(w)
		return
	}

	if user.ID == util.UserBrief(r).ID {
		errorData(w, "You can't demote yourself to tester rank!", 400)
		return
	}

	if err := s.base.AddContestTester(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Added contest tester")
}

func (s *API) stripContestAccess(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		UserID int `json:"user_id"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID == util.UserBrief(r).ID {
		errorData(w, "You can't strip your own access!", 400)
		return
	}

	if err := s.base.StripContestAccess(r.Context(), util.Contest(r).ID, args.UserID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Stripped contest access")
}

func (s *API) getContestAccessControl(w http.ResponseWriter, r *http.Request) {
	returnData(w, struct {
		Editors []*kilonova.UserBrief `json:"editors"`
		Testers []*kilonova.UserBrief `json:"testers"`
	}{
		Editors: util.Contest(r).Editors,
		Testers: util.Contest(r).Testers,
	})
}

func (s *API) contestAnnouncements(w http.ResponseWriter, r *http.Request) {
	announcements, err := s.base.ContestAnnouncements(r.Context(), util.Contest(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, announcements)
}

func (s *API) createContestAnnouncement(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Text string `json:"text"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Text == "" {
		errorData(w, "No announcement text supplied", 400)
		return
	}

	_, err := s.base.CreateContestAnnouncement(r.Context(), util.Contest(r).ID, args.Text)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Created announcement")
}

func (s *API) updateContestAnnouncement(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID   int    `json:"id"`
		Text string `json:"text"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	announcement, err := s.base.ContestAnnouncement(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if announcement.ContestID != util.Contest(r).ID {
		errorData(w, "Contest announcement must be from contest", 400)
		return
	}

	if err := s.base.UpdateContestAnnouncement(r.Context(), args.ID, args.Text); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Updated announcement")
}

func (s *API) deleteContestAnnouncement(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID int `json:"id"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	announcement, err := s.base.ContestAnnouncement(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if announcement.ContestID != util.Contest(r).ID {
		errorData(w, "Contest announcement must be from contest", 400)
		return
	}

	if err := s.base.DeleteContestAnnouncement(r.Context(), announcement.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Removed announcement")
}

func (s *API) contestUserQuestions(w http.ResponseWriter, r *http.Request) {
	qs, err := s.base.ContestUserQuestions(r.Context(), util.Contest(r).ID, util.UserBrief(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, qs)
}

func (s *API) contestAllQuestions(w http.ResponseWriter, r *http.Request) {
	qs, err := s.base.ContestQuestions(r.Context(), util.Contest(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, qs)
}

func (s *API) askContestQuestion(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Text string `json:"text"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Text == "" {
		errorData(w, "No question text supplied", 400)
		return
	}

	if _, err := s.base.CreateContestQuestion(r.Context(), util.Contest(r).ID, util.UserBrief(r).ID, args.Text); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Asked question")
}

func (s *API) answerContestQuestion(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID   int    `json:"questionID"`
		Text string `json:"text"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Text == "" {
		errorData(w, "No question response text supplied", 400)
		return
	}

	question, err := s.base.ContestQuestion(r.Context(), args.ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	if question.ContestID != util.Contest(r).ID {
		errorData(w, "Contest question must be from contest", 400)
		return
	}

	if err := s.base.AnswerContestQuestion(r.Context(), args.ID, args.Text); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Answered question")
}

func (s *API) registerForContest(w http.ResponseWriter, r *http.Request) {
	if err := s.base.RegisterContestUser(r.Context(), util.Contest(r).ID, util.UserBrief(r).ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Registered for contest")
}

func (s *API) checkRegistration(w http.ResponseWriter, r *http.Request) {
	reg, err := s.base.ContestRegistration(r.Context(), util.Contest(r).ID, util.UserBrief(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, reg)
}

func (s *API) contestRegistrations(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Limit <= 0 {
		args.Limit = 50
	}

	regs, err := s.base.ContestRegistrations(r.Context(), util.Contest(r).ID, args.Limit, args.Offset)
	if err != nil {
		err.WriteError(w)
		return
	}

	cnt, err := s.base.ContestRegistrationCount(r.Context(), util.Contest(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	ids := []int{}

	for _, reg := range regs {
		ids = append(ids, reg.UserID)
	}

	users, err := s.base.UsersBrief(r.Context(), kilonova.UserFilter{
		IDs: ids,
	})
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Users []*kilonova.UserBrief `json:"users"`

		Count int `json:"total_count"`
	}{Users: users, Count: cnt})
}
