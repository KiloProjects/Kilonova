package api

import (
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/davecgh/go-spew/spew"
	"go.uber.org/zap"
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
	r.ParseForm()
	var args struct {
		ID  int
		Set bool
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
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
	var upd = make(map[string]bool)
	r.ParseForm()
	for k, val := range r.Form {
		if len(val) > 0 {
			bl, err := strconv.ParseBool(val[0])
			if err != nil {
				errorData(w, err, 400)
				return
			}
			upd[k] = bl
		}
	}

	for k, v := range upd {
		flg, ok := config.GetFlag[bool](k)
		if !ok {
			zap.S().Warnf("Flag %q not found", k)
			continue
		}
		flg.Update(v)
	}
	returnData(w, "Updated flags. Some changes may only apply after a restart")
}

func (s *API) addDonation(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var args struct {
		kilonova.Donation
		Username *string `json:"username"`
	}
	if err := decoder.Decode(&args, r.Form); err != nil {
		spew.Dump(err)
		errorData(w, "Invalid request parameters", 400)
		return
	}

	spew.Dump(args)

	if args.Username != nil && *args.Username != "" {
		user, err := s.base.UserBriefByName(r.Context(), *args.Username)
		if err != nil {
			err.WriteError(w)
			return
		}
		args.User = user
	}
	if err := s.base.AddDonation(r.Context(), &args.Donation); err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, args.ID)
}
