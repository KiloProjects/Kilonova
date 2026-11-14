package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
)

func (s *API) setAdmin(w http.ResponseWriter, r *http.Request) {
	var args struct {
		ID  int
		Set bool
	}
	if err := parseRequest(r, &args); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if args.ID <= 0 {
		errorData(w, "Invalid ID", 400)
		return
	}

	user, err := s.base.UserBrief(r.Context(), args.ID)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.SetAdmin(r.Context(), user, args.Set); err != nil {
		statusError(w, err)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added admin")
		return
	}
	returnData(w, "Succesfully removed admin")
}

func (s *API) setProposer(w http.ResponseWriter, r *http.Request) {
	var args struct {
		ID  int
		Set bool
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if args.ID <= 0 {
		errorData(w, "Invalid ID", 400)
		return
	}

	user, err := s.base.UserBrief(r.Context(), args.ID)
	if err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.SetProposer(r.Context(), user, args.Set); err != nil {
		statusError(w, err)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added proposer")
		return
	}
	returnData(w, "Succesfully removed proposer")
}

func (s *API) addDonation(w http.ResponseWriter, r *http.Request) {
	var args struct {
		kilonova.Donation
		Username *string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if args.Username != nil && *args.Username != "" {
		user, err := s.base.UserBriefByName(r.Context(), *args.Username)
		if err != nil {
			args.RealName = *args.Username
		} else {
			args.User = user
		}
	}
	if err := s.base.AddDonation(r.Context(), &args.Donation); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, args.ID)
}

func (s *API) endSubscription(w http.ResponseWriter, r *http.Request) {
	var args struct {
		ID int
	}
	if err := parseRequest(r, &args); err != nil {
		statusError(w, err)
		return
	}

	if err := s.base.CancelSubscription(r.Context(), args.ID); err != nil {
		statusError(w, err)
		return
	}
	returnData(w, args.ID)
}
