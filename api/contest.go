package api

import (
	"context"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/util"
	"go.uber.org/zap"
)

func (s *API) createContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Name string               `json:"name"`
		Type kilonova.ContestType `json:"type"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	id, err := s.base.CreateContest(r.Context(), args.Name, args.Type, util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, id)
}

func (s *API) updateContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args kilonova.ContestUpdate
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, 500)
		return
	}

	if args.Type != kilonova.ContestTypeNone && args.Type != util.Contest(r).Type && !s.base.IsAdmin(util.UserBrief(r)) {
		errorData(w, "You aren't allowed to change contest type!", 400)
		return
	}
	st := util.Contest(r).StartTime
	et := util.Contest(r).EndTime
	if args.StartTime != nil {
		st = *args.StartTime
	}
	if args.EndTime != nil {
		et = *args.EndTime
	}
	if !st.Before(et) {
		errorData(w, "Start time must be before end time.", 400)
		return
	}

	if err := s.base.UpdateContest(r.Context(), util.Contest(r).ID, args); err != nil {
		err.WriteError(w)
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
		return
	}

	list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r), util.Contest(r).Type == kilonova.ContestTypeOfficial)
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

func (s *API) getContest(ctx context.Context, args struct{}) (*kilonova.Contest, *kilonova.StatusError) {
	return util.ContestContext(ctx), nil
}

func (s *API) getContestProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := s.base.ContestProblems(r.Context(), util.Contest(r), util.UserBrief(r))
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, pbs)
}

func (s *API) contestLeaderboard(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Frozen bool `json:"frozen"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		http.Error(w, "Can't decode parameters", 400)
		return
	}
	// This is assumed to be called from a context in which
	// IsContestVisible is already true
	if !(util.Contest(r).PublicLeaderboard || s.base.IsContestEditor(util.UserBrief(r), util.Contest(r))) {
		errorData(w, "You are not allowed to view the leaderboard", 400)
		return
	}

	ld, err := s.base.ContestLeaderboard(
		r.Context(), util.Contest(r),
		s.base.UserContestFreezeTime(util.UserBrief(r), util.Contest(r), args.Frozen),
	)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, ld)
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

func (s *API) contestAnnouncements(ctx context.Context, args struct{}) ([]*kilonova.ContestAnnouncement, *kilonova.StatusError) {
	return s.base.ContestAnnouncements(ctx, util.ContestContext(ctx).ID)
}

func (s *API) createContestAnnouncement(ctx context.Context, args struct {
	Text string `json:"text"`
}) *kilonova.StatusError {
	if args.Text == "" {
		return kilonova.Statusf(400, "No announcement text supplied")
	}

	_, err := s.base.CreateContestAnnouncement(ctx, util.ContestContext(ctx).ID, args.Text)
	return err
}

func (s *API) updateContestAnnouncement(ctx context.Context, args struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}) *kilonova.StatusError {
	announcement, err := s.base.ContestAnnouncement(ctx, args.ID)
	if err != nil {
		return err
	}

	if announcement.ContestID != util.ContestContext(ctx).ID {
		return kilonova.Statusf(400, "Contest announcement must be from contest")
	}

	return s.base.UpdateContestAnnouncement(ctx, args.ID, args.Text)
}

func (s *API) deleteContestAnnouncement(ctx context.Context, args struct {
	ID int `json:"id"`
}) *kilonova.StatusError {
	announcement, err := s.base.ContestAnnouncement(ctx, args.ID)
	if err != nil {
		return err
	}

	if announcement.ContestID != util.ContestContext(ctx).ID {
		return kilonova.Statusf(400, "Contest announcement must be from contest")
	}

	return s.base.DeleteContestAnnouncement(ctx, announcement.ID)
}

func (s *API) contestUserQuestions(ctx context.Context, args struct{}) ([]*kilonova.ContestQuestion, *kilonova.StatusError) {
	return s.base.ContestUserQuestions(ctx, util.ContestContext(ctx).ID, util.UserBriefContext(ctx).ID)
}

func (s *API) contestAllQuestions(ctx context.Context, args struct{}) ([]*kilonova.ContestQuestion, *kilonova.StatusError) {
	return s.base.ContestQuestions(ctx, util.ContestContext(ctx).ID)
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

func (s *API) acceptContestInvitation(ctx context.Context, args struct {
	InviteID string `json:"invite_id"`
}) *kilonova.StatusError {
	inv, err := s.base.ContestInvitation(ctx, args.InviteID)
	if err != nil {
		return err
	}
	if inv.Expired {
		return kilonova.Statusf(400, "Invite expired")
	}
	contest, err := s.base.Contest(ctx, inv.ContestID)
	if err != nil {
		return err
	}
	if contest.Ended() {
		return kilonova.Statusf(400, "Contest ended")
	}
	if !contest.RegisterDuringContest && contest.Running() {
		return kilonova.Statusf(400, "Cannot register while contest is running")
	}
	return s.base.RegisterContestUser(ctx, contest, util.UserBriefContext(ctx).ID, &inv.ID, true)
}

func (s *API) updateContestInvitation(ctx context.Context, args struct {
	InviteID string `json:"invite_id"`
	Expired  bool   `json:"expired"`
}) *kilonova.StatusError {
	inv, err := s.base.ContestInvitation(ctx, args.InviteID)
	if err != nil {
		return err
	}
	contest, err := s.base.Contest(ctx, inv.ContestID)
	if err != nil {
		return err
	}
	if !s.base.IsContestEditor(util.UserBriefContext(ctx), contest) {
		return kilonova.Statusf(400, "Only contest editors can update the invitation")
	}
	return s.base.UpdateContestInvitation(ctx, inv.ID, args.Expired)
}

func (s *API) registerForContest(w http.ResponseWriter, r *http.Request) {
	if err := s.base.RegisterContestUser(r.Context(), util.Contest(r), util.UserBrief(r).ID, nil, false); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Registered for contest")
}

func (s *API) startContestRegistration(w http.ResponseWriter, r *http.Request) {
	if err := s.base.StartContestRegistration(r.Context(), util.Contest(r), util.UserBrief(r).ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Started contest registration.")
}

func (s *API) forceRegisterForContest(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"name"`
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

	if _, err := s.base.ContestRegistration(r.Context(), util.Contest(r).ID, user.ID); err == nil {
		errorData(w, "User is already registered", 400)
		return
	}

	if err := s.base.RegisterContestUser(r.Context(), util.Contest(r), user.ID, nil, true); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, "Force registered user for contest")
}

func (s *API) checkRegistration(ctx context.Context, args struct{}) (*kilonova.ContestRegistration, *kilonova.StatusError) {
	return s.base.ContestRegistration(ctx, util.ContestContext(ctx).ID, util.UserBriefContext(ctx).ID)
}

func (s *API) stripContestRegistration(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		Username string `json:"name"`
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

	if err := s.base.KickUserFromContest(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, "Kicked user")
}

type regRez struct {
	User *kilonova.UserBrief           `json:"user"`
	Reg  *kilonova.ContestRegistration `json:"registration"`
}

func (s *API) contestRegistrations(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		FuzzyName    *string `json:"name_fuzzy"`
		InvitationID *string `json:"invitation_id"`

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

	regs, err := s.base.ContestRegistrations(r.Context(), util.Contest(r).ID, args.FuzzyName, args.InvitationID, args.Limit, args.Offset)
	if err != nil {
		err.WriteError(w)
		return
	}

	cnt, err := s.base.ContestRegistrationCount(r.Context(), util.Contest(r).ID)
	if err != nil {
		err.WriteError(w)
		return
	}

	regMap := make(map[int]*kilonova.ContestRegistration)
	for _, reg := range regs {
		regMap[reg.UserID] = reg
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

	var rez = make([]regRez, 0, len(users))
	if len(users) != len(regs) {
		zap.S().Warn("mismatched user and reg length")
	}

	for _, user := range users {
		user := user
		val, ok := regMap[user.ID]
		if !ok {
			zap.S().Warnf("Couldn't find user %d in registrations", user.ID)
		}
		rez = append(rez, regRez{User: user, Reg: val})
	}

	returnData(w, struct {
		Registrations []regRez `json:"registrations"`

		Count int `json:"total_count"`
	}{Registrations: rez, Count: cnt})
}
