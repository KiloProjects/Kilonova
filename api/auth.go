package api

import (
	"net/http"

	"github.com/KiloProjects/kilonova"
	"go.uber.org/zap"
)

/*
	NOTE: Session expires after 30 days
	Cookie should look like this:
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    sid,
		Path:     "/",
		HttpOnly: false,
		Expires:  time.Now().Add(time.Hour * 24 * 30),
	}
*/

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string
		Email    string
		Password string
		Language string
		Theme    string
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	uid, status := s.base.Signup(r.Context(), auth.Email, auth.Username, auth.Password, auth.Language, kilonova.PreferredTheme(auth.Theme))
	if status != nil {
		status.WriteError(w)
		return
	}

	sid, err1 := s.base.CreateSession(r.Context(), uid)
	if err1 != nil {
		err1.WriteError(w)
		return
	}
	returnData(w, sid)
}

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	var auth struct {
		Username string
		Password string
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	uid, status := s.base.Login(r.Context(), auth.Username, auth.Password)
	if status != nil {
		status.WriteError(w)
		return
	}

	sid, err1 := s.base.CreateSession(r.Context(), uid)
	if err1 != nil {
		err1.WriteError(w)
		return
	}
	returnData(w, sid)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	h := getAuthHeader(r)
	if h == "" {
		errorData(w, "You are already logged out!", 400)
		return
	}
	s.base.RemoveSession(r.Context(), h)
	returnData(w, "Logged out")
}

func (s *API) extendSession(w http.ResponseWriter, r *http.Request) {
	h := getAuthHeader(r)
	if h == "" {
		zap.S().Warn("Empty session on endpoint that must be authed")
		return
	}
	exp, err := s.base.ExtendSession(r.Context(), h)
	if err != nil {
		err.WriteError(w)
		return
	}
	returnData(w, exp)
}
