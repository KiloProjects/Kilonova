package api

import (
	"log/slog"
	"net/http"
)

func (s *API) login(w http.ResponseWriter, r *http.Request) {
	var auth struct {
		Username string
		Password string
		// Persistent chooses a 30 day session over a 24 hour one.
		// Left out, it is true, which is what this endpoint always did.
		Persistent *bool `json:"persistent"`
	}

	if err := parseRequest(r, &auth); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	user, status := s.base.Login(r.Context(), auth.Username, auth.Password)
	if status != nil {
		statusError(w, status)
		return
	}

	if user.LockedLogin && !user.Admin {
		// Lockout but don't lockout admins
		errorData(w, "Login for this account has been restricted by an administrator", 401)
		return
	}

	sid, err := s.base.CreateSession(r.Context(), user.ID, auth.Persistent == nil || *auth.Persistent)
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, sid)
}

func (s *API) logout(w http.ResponseWriter, r *http.Request) {
	s.base.RemoveSession(r.Context(), getAuthHeader(r))
	returnData(w, "Logged out")
}

func (s *API) extendSession(w http.ResponseWriter, r *http.Request) {
	h := getAuthHeader(r)
	if h == "" {
		slog.DebugContext(r.Context(), "Empty session on endpoint that must be authed")
		returnData(w, nil)
		return
	}
	sess, err := s.base.ExtendSession(r.Context(), h)
	if err != nil {
		statusError(w, err)
		return
	}
	returnData(w, sess.ExpiresAt)
}
