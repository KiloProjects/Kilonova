package api

import (
	"cmp"
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/domain/user"
	"github.com/KiloProjects/kilonova/internal/util"
	"github.com/KiloProjects/kilonova/util/slicealg"
)

func (s *API) createContest(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Name string               `json:"name"`
		Type kilonova.ContestType `json:"type"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 500)
		return
	}

	id, err := s.base.CreateContest(r.Context(), args.Name, args.Type, util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}

	returnData(w, id)
}

func (s *API) updateContest(w http.ResponseWriter, r *http.Request) {
	var args kilonova.ContestUpdate
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 500)
		return
	}

	if !util.UserBrief(r).IsAdmin() {
		if args.Type != kilonova.ContestTypeNone && args.Type != util.Contest(r).Type {
			errorData(w, "You aren't allowed to change contest type!", 400)
			return
		}
		if args.WhitelistEnabled != nil && *args.WhitelistEnabled != util.Contest(r).WhitelistEnabled {
			errorData(w, "You aren't allowed to change whitelist status!", 400)
			return
		}
		if args.IPManagementEnabled != nil && *args.IPManagementEnabled != util.Contest(r).IPManagementEnabled {
			errorData(w, "You aren't allowed to change whitelist status!", 400)
			return
		}
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
		statusError(w, err)
		return
	}

	returnData(w, "Contest updated")
}

func (s *API) updateContestProblems(w http.ResponseWriter, r *http.Request) {
	var args struct {
		List []int `json:"list"`
	}
	if err := parseJSONBody(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if args.List == nil {
		errorData(w, "You must specify a list of problems", 400)
		return
	}

	list, err := s.filterProblems(r.Context(), args.List, util.UserBrief(r), util.Contest(r).Type == kilonova.ContestTypeOfficial, true)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.UpdateContestProblems(r.Context(), util.Contest(r).ID, list); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Updated contest problems")
}

func (s *API) getContest(ctx context.Context, _ struct{}) (*kilonova.Contest, error) {
	return util.ContestContext(ctx), nil
}

func (s *API) getContestProblems(w http.ResponseWriter, r *http.Request) {
	pbs, err := s.base.ContestProblems(r.Context(), util.Contest(r), util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, pbs)
}

type subCountResult struct {
	Limited   bool `json:"limited"`
	Remaining int  `json:"remaining"`
}

func (s *API) getRemainingSubmissionCount(w http.ResponseWriter, r *http.Request) {
	var args struct {
		UserID int `json:"user_id"`
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}
	user := util.UserBrief(r)
	//if s.base.IsContestEditor(util.UserBrief(r), util.Contest(r)) {
	//
	//}
	pbs, err := s.base.ContestProblems(r.Context(), util.Contest(r), util.UserBrief(r))
	if err != nil {
		statusError(w, err)
		return
	}
	var remainingCount = make(map[int]subCountResult)
	for _, pb := range pbs {
		cnt, limited, err := s.base.RemainingSubmissionCount(r.Context(), util.Contest(r), pb.ID, util.UserBrief(r).ID)
		if err != nil {
			statusError(w, err)
			return
		}
		remainingCount[pb.ID] = subCountResult{
			Limited:   limited,
			Remaining: cnt,
		}
	}
	returnData(w, struct {
		UserID int                    `json:"user_id"`
		Counts map[int]subCountResult `json:"counts"`
	}{
		UserID: user.ID,
		Counts: remainingCount,
	})
}

type contestLeaderboardParams struct {
	Frozen bool `json:"frozen"`

	Generated *bool `json:"generated_acc"`
}

func (s *API) leaderboard(ctx context.Context, contest *kilonova.Contest, lookingUser *kilonova.UserBrief, args *contestLeaderboardParams) (*kilonova.ContestLeaderboard, error) {
	if !s.base.CanViewContestLeaderboard(lookingUser, contest) {
		return nil, kilonova.Statusf(400, "Leaderboard for this contest is not available")
	}

	return s.base.ContestLeaderboard(
		ctx, contest,
		s.base.UserContestFreezeTime(lookingUser, contest, args.Frozen),
		args.Generated,
	)
}

func (s *API) contestLeaderboard(w http.ResponseWriter, r *http.Request) {
	var args contestLeaderboardParams
	if err := parseRequest(r, &args); err != nil {
		http.Error(w, "Can't decode parameters", 400)
		return
	}
	ld, err := s.leaderboard(r.Context(), util.Contest(r), util.UserBrief(r), &args)
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, ld)
}

func (s *API) addContestEditor(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.AddContestEditor(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Added contest editor")
}

func (s *API) addContestTester(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if user.ID == util.UserBrief(r).ID {
		errorData(w, "You can't demote yourself to tester rank!", 400)
		return
	}

	if err := s.base.AddContestTester(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Added contest tester")
}

func (s *API) stripContestAccess(w http.ResponseWriter, r *http.Request) {
	var args struct {
		UserID int `json:"user_id"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.UserID == util.UserBrief(r).ID {
		errorData(w, "You can't strip your own access!", 400)
		return
	}

	if err := s.base.StripContestAccess(r.Context(), util.Contest(r).ID, args.UserID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Stripped contest access")
}

func (s *API) contestAnnouncements(ctx context.Context, _ struct{}) ([]*kilonova.ContestAnnouncement, error) {
	return s.base.ContestAnnouncements(ctx, util.ContestContext(ctx).ID)
}

func (s *API) createContestAnnouncement(ctx context.Context, args struct {
	Text string `json:"text"`
}) error {
	if args.Text == "" {
		return kilonova.Statusf(400, "No announcement text supplied")
	}

	_, err := s.base.CreateContestAnnouncement(ctx, util.ContestContext(ctx).ID, args.Text)
	return err
}

func (s *API) updateContestAnnouncement(ctx context.Context, args struct {
	ID   int    `json:"id"`
	Text string `json:"text"`
}) error {
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
}) error {
	announcement, err := s.base.ContestAnnouncement(ctx, args.ID)
	if err != nil {
		return err
	}

	if announcement.ContestID != util.ContestContext(ctx).ID {
		return kilonova.Statusf(400, "Contest announcement must be from contest")
	}

	if util.ContestContext(ctx).Ended() {
		return kilonova.Statusf(400, "Ended contests should not be modified")
	}

	return s.base.DeleteContestAnnouncement(ctx, announcement.ID)
}

func (s *API) contestUserQuestions(ctx context.Context, _ struct{}) ([]*kilonova.ContestQuestion, error) {
	if user.UserBriefContext(ctx) == nil {
		return []*kilonova.ContestQuestion{}, nil
	}
	return s.base.ContestUserQuestions(ctx, util.ContestContext(ctx).ID, user.UserBriefContext(ctx).ID)
}

func (s *API) contestAllQuestions(ctx context.Context, _ struct{}) ([]*kilonova.ContestQuestion, error) {
	return s.base.ContestQuestions(ctx, util.ContestContext(ctx).ID)
}

func (s *API) askContestQuestion(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Text string `json:"text"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Text == "" {
		errorData(w, "No question text supplied", 400)
		return
	}

	if _, err := s.base.CreateContestQuestion(r.Context(), util.Contest(r), util.UserBrief(r).ID, args.Text); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Asked question")
}

func (s *API) answerContestQuestion(w http.ResponseWriter, r *http.Request) {
	var args struct {
		ID   int    `json:"questionID"`
		Text string `json:"text"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Text == "" {
		errorData(w, "No question response text supplied", 400)
		return
	}

	question, err := s.base.ContestQuestion(r.Context(), args.ID)
	if err != nil {
		statusError(w, err)
		return
	}

	if question.ContestID != util.Contest(r).ID {
		errorData(w, "Contest question must be from contest", 400)
		return
	}

	if err := s.base.AnswerContestQuestion(r.Context(), args.ID, args.Text); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Answered question")
}

func (s *API) acceptContestInvitation(ctx context.Context, args struct {
	InviteID string `json:"invite_id"`
}) error {
	inv, err := s.base.ContestInvitation(ctx, args.InviteID)
	if err != nil {
		return err
	}
	if inv.Expired {
		return kilonova.Statusf(400, "Invite expired")
	}
	if inv.MaxCount != nil && *inv.MaxCount <= inv.RedeemCount {
		return kilonova.Statusf(400, "Invite limit reached")
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
	return s.base.RegisterContestUser(ctx, contest, user.UserBriefContext(ctx).ID, &inv.ID, true)
}

func (s *API) updateContestInvitation(ctx context.Context, args struct {
	InviteID string `json:"invite_id"`
	Expired  bool   `json:"expired"`
}) error {
	inv, err := s.base.ContestInvitation(ctx, args.InviteID)
	if err != nil {
		return err
	}
	contest, err := s.base.Contest(ctx, inv.ContestID)
	if err != nil {
		return err
	}
	if !contest.IsEditor(user.UserBriefContext(ctx)) {
		return kilonova.Statusf(400, "Only contest editors can update the invitation")
	}
	return s.base.UpdateContestInvitation(ctx, inv.ID, args.Expired)
}

func (s *API) registerForContest(w http.ResponseWriter, r *http.Request) {
	if err := s.base.RegisterContestUser(r.Context(), util.Contest(r), util.UserBrief(r).ID, nil, false); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, "Registered for contest")
}

func (s *API) startContestRegistration(w http.ResponseWriter, r *http.Request) {
	if err := s.base.StartContestRegistration(r.Context(), util.Contest(r), util.UserBrief(r).ID); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, "Started contest registration.")
}

func (s *API) forceRegisterForContest(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"name"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if _, err := s.base.ContestRegistration(r.Context(), util.Contest(r).ID, user.ID); err == nil {
		errorData(w, "User is already registered", 400)
		return
	}

	if err := s.base.RegisterContestUser(r.Context(), util.Contest(r), user.ID, nil, true); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, "Force registered user for contest")
}

func (s *API) checkRegistration(ctx context.Context, _ struct{}) (*kilonova.ContestRegistration, error) {
	return s.base.ContestRegistration(ctx, util.ContestContext(ctx).ID, user.UserBriefContext(ctx).ID)
}

func (s *API) stripContestRegistration(w http.ResponseWriter, r *http.Request) {
	var args struct {
		Username string `json:"name"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	user, err := s.base.UserBriefByName(r.Context(), args.Username)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.KickUserFromContest(r.Context(), util.Contest(r).ID, user.ID); err != nil {
		statusError(w, err)
		return
	}

	returnData(w, "Kicked user")
}

type regRez struct {
	User *kilonova.UserBrief           `json:"user"`
	Reg  *kilonova.ContestRegistration `json:"registration"`
}

func (s *API) contestRegistrations(w http.ResponseWriter, r *http.Request) {
	var args struct {
		FuzzyName    *string `json:"name_fuzzy"`
		InvitationID *string `json:"invitation_id"`

		Limit  uint64 `json:"limit"`
		Offset uint64 `json:"offset"`
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, 400)
		return
	}

	if args.Limit <= 0 {
		args.Limit = 50
	}

	regs, err := s.base.ContestRegistrations(r.Context(), util.Contest(r).ID, args.FuzzyName, args.InvitationID, args.Limit, args.Offset)
	if err != nil {
		statusError(w, err)
		return
	}

	cnt, err := s.base.ContestRegistrationCount(r.Context(), util.Contest(r).ID)
	if err != nil {
		statusError(w, err)
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
		statusError(w, err)
		return
	}

	var rez = make([]regRez, 0, len(users))
	if len(users) != len(regs) {
		slog.WarnContext(r.Context(), "mismatched user and reg length", slog.Int("users_len", len(users)), slog.Int("regs_len", len(regs)))
	}

	for _, user := range users {
		val, ok := regMap[user.ID]
		if !ok {
			slog.WarnContext(r.Context(), "Couldn't find user in registrations", slog.Any("user", user))
		}
		rez = append(rez, regRez{User: user, Reg: val})
	}

	returnData(w, struct {
		Registrations []regRez `json:"registrations"`

		Count int `json:"total_count"`
	}{Registrations: rez, Count: cnt})
}

func (s *API) runMOSS(ctx context.Context, _ struct{}) error {
	if util.ContestContext(ctx).Type != kilonova.ContestTypeOfficial {
		return kilonova.Statusf(400, "MOSS can't run on virtual contests, for now")
	}
	return s.base.RunMOSS(context.WithoutCancel(ctx), util.ContestContext(ctx))
}

///

type ContestGetInput struct {
	Body *struct {
		Type    kilonova.ContestType `json:"type" required:"false" oneOf:"official,virtual"`
		Future  bool                 `json:"future" required:"false"`
		Running bool                 `json:"running" required:"false"`
		Ended   bool                 `json:"ended" required:"false"`

		Limit  int `json:"limit" maximum:"50" required:"false"`
		Offset int `json:"offset" required:"false"`
	}
}

type apiContest struct {
	ID        int                   `json:"id"`
	CreatedAt time.Time             `json:"created_at"`
	Name      string                `json:"name"`
	Editors   []*kilonova.UserBrief `json:"editors"`
	Testers   []*kilonova.UserBrief `json:"testers"`

	Description string `json:"description"`

	// PublicJoin indicates whether a user can freely join a contest
	// or he needs to be manually added
	PublicJoin bool `json:"public_join"`

	// RegisterDuringContest indicates whether a user can join a contest while it's running
	// It is useless without PublicJoin set to true
	RegisterDuringContest bool `json:"register_during_contest"`

	// Visible indicates whether a contest can be seen by others
	// Contestants may be able to see the contest
	Visible bool `json:"hidden"`

	// PublicLeaderboard controls whether the contest's leaderboard
	// is viewable by everybody or just admins
	PublicLeaderboard bool `json:"public_leaderboard"`

	LeaderboardStyle      kilonova.LeaderboardType `json:"leaderboard_style"`
	LeaderboardFreeze     *time.Time               `json:"leaderboard_freeze"`
	ICPCSubmissionPenalty int                      `json:"icpc_submission_penalty"`

	LeaderboardAdvancedFilter bool `json:"leaderboard_advanced_filter"`

	SubmissionCooldown time.Duration `json:"submission_cooldown"`
	QuestionCooldown   time.Duration `json:"question_cooldown"`

	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`

	// PerUserTime records the number of seconds a user has in an USACO-style participation
	// Setting it to 0 will make contests behave "normally"
	PerUserTime int `json:"per_user_time"`

	Type kilonova.ContestType `json:"type"`

	// MaxSubs is the maximum number of submissions
	// that someone is allowed to send to a problem during a contest.
	// Any number < 0 means no limit
	MaxSubs int `json:"max_subs"`
}

func contestFromDomain(c *kilonova.Contest) *apiContest {
	return &apiContest{
		ID:                        c.ID,
		CreatedAt:                 c.CreatedAt,
		Name:                      c.Name,
		Editors:                   c.Editors,
		Testers:                   c.Testers,
		Description:               c.Description,
		PublicJoin:                c.PublicJoin,
		RegisterDuringContest:     c.RegisterDuringContest,
		Visible:                   c.Visible,
		PublicLeaderboard:         c.PublicLeaderboard,
		LeaderboardStyle:          c.LeaderboardStyle,
		LeaderboardFreeze:         c.LeaderboardFreeze,
		ICPCSubmissionPenalty:     c.ICPCSubmissionPenalty,
		LeaderboardAdvancedFilter: c.LeaderboardAdvancedFilter,
		SubmissionCooldown:        c.SubmissionCooldown,
		QuestionCooldown:          c.QuestionCooldown,
		StartTime:                 c.StartTime,
		EndTime:                   c.EndTime,
		PerUserTime:               c.PerUserTime,
		Type:                      c.Type,
		MaxSubs:                   c.MaxSubs,
	}
}

type contestSearchResult struct {
	Contests []*apiContest `json:"contests"`
}

type ContestGetOutput struct {
	Body contestSearchResult
}

func (s *API) contestGet(ctx context.Context, input *ContestGetInput) (*ContestGetOutput, error) {
	var args kilonova.ContestFilter
	if input.Body != nil {
		args = kilonova.ContestFilter{
			Future:  input.Body.Future,
			Running: input.Body.Running,
			Ended:   input.Body.Ended,
			Type:    input.Body.Type,
			Limit:   cmp.Or(max(min(input.Body.Limit, 50), 0), 25),
			Offset:  input.Body.Offset,
		}
	}
	args.Look = true
	args.LookingUser = user.UserBriefContext(ctx)

	contests, err := s.base.Contests(ctx, args)
	if err != nil {
		return nil, err
	}
	return &ContestGetOutput{
		Body: contestSearchResult{Contests: slicealg.Map(contests, contestFromDomain)},
	}, nil
}

type ContestSingleGetOutput struct {
	Body *apiContest
}

func (s *API) contestSingleGet(ctx context.Context, _ *struct{}) (*ContestSingleGetOutput, error) {
	return &ContestSingleGetOutput{contestFromDomain(util.ContestContext(ctx))}, nil
}
