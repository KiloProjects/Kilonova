package api

import (
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
)

func (s *API) setAdmin(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		ID  int
		Set bool
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	if err := s.base.SetAdmin(r.Context(), args.ID, args.Set); err != nil {
		err.WriteError(w)
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
		err.WriteError(w)
		return
	}

	if err := s.base.SetProposer(r.Context(), args.ID, args.Set); err != nil {
		err.WriteError(w)
		return
	}

	if args.Set {
		returnData(w, "Succesfully added proposer")
		return
	}
	returnData(w, "Succesfully removed proposer")
}

func (s *API) getAllUsers(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var query kilonova.UserFilter
	if err := decoder.Decode(&query, r.Form); err != nil {
		errorData(w, "Invalid request parameters", 400)
		return
	}
	rez, err := s.base.UsersBrief(r.Context(), query)
	if err != nil {
		err.WriteError(w)
		return
	}

	cnt, err := s.base.CountUsers(r.Context(), query)
	if err != nil {
		err.WriteError(w)
		return
	}

	returnData(w, struct {
		Users []*kilonova.UserBrief `json:"users"`

		Count int `json:"total_count"`
	}{Users: rez, Count: cnt})
}

func (s *API) updateBoolFlags(w http.ResponseWriter, r *http.Request) {
	var args struct {
		BoolFlags   map[string]bool   `json:"bool_flags"`
		StringFlags map[string]string `json:"string_flags"`
		IntFlags    map[string]int    `json:"int_flags"`
	}
	if err := parseJSONBody(r, &args); err != nil {
		err.WriteError(w)
		return
	}
	for k, v := range args.BoolFlags {
		flg, ok := config.GetFlag[bool](k)
		if !ok {
			slog.Warn("Flag not found", slog.String("name", k))
			continue
		}
		flg.Update(v)
	}
	for k, v := range args.StringFlags {
		flg, ok := config.GetFlag[string](k)
		if !ok {
			slog.Warn("Flag not found", slog.String("name", k))
			continue
		}
		flg.Update(v)
	}
	for k, v := range args.IntFlags {
		flg, ok := config.GetFlag[int](k)
		if !ok {
			slog.Warn("Flag not found", slog.String("name", k))
			continue
		}
		flg.Update(v)
	}
	returnData(w, "Updated flags. Some changes may only apply after a restart")
}

func (s *API) addDonation(w http.ResponseWriter, r *http.Request) {
	var args struct {
		kilonova.Donation
		Username *string `json:"username"`
	}
	if err := parseRequest(r, &args); err != nil {
		err.WriteError(w)
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
		err.WriteError(w)
		return
	}
	returnData(w, args.ID)
}

func (s *API) endSubscription(w http.ResponseWriter, r *http.Request) {
	var args struct {
		ID int
	}
	if err := parseRequest(r, &args); err != nil {
		err.WriteError(w)
		return
	}

	if err := s.base.CancelSubscription(r.Context(), args.ID); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, args.ID)
}
