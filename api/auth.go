package api

import (
	"net/http"
)

/*
	NOTE: Session expires after 30 days
	Cookie should look like this:
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    sid,
		Path:     "/",
		HttpOnly: false,
		SameSite: http.SameSiteDefaultMode,
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
	}

	if err := decoder.Decode(&auth, r.Form); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	uid, status := s.base.Signup(r.Context(), auth.Email, auth.Username, auth.Password, auth.Language)
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
