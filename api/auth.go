package api

import (
	"log/slog"
	"net/http"

	"github.com/KiloProjects/kilonova"
)

func (s *API) signup(w http.ResponseWriter, r *http.Request) {
	s.signupLock.Lock()
	defer s.signupLock.Unlock()
	var auth struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Language string `json:"language"`
		Theme    string `json:"theme"`

		CaptchaID       string `json:"captcha_id"`
		CaptchaResponse string `json:"captcha_response"`
	}

	if err := parseRequest(r, &auth); err != nil {
		errorData(w, err, http.StatusBadRequest)
		return
	}

	ip, ua := s.base.GetRequestInfo(r)
	if s.base.MustSolveCaptcha(r.Context(), ip) {
		if auth.CaptchaID == "" || auth.CaptchaResponse == "" {
			errorData(w, struct {
				ID  string `json:"captcha_id"`
				Key string `json:"translation_key"`
			}{
				ID:  s.base.NewCaptchaID(),
				Key: "auth.captcha.must_solve",
			}, http.StatusPreconditionRequired)
			return
		}
		if !s.base.CheckCaptcha(auth.CaptchaID, auth.CaptchaResponse) {
			errorData(w, struct {
				ID  string `json:"captcha_id"`
				Key string `json:"translation_key"`
			}{
				ID:  s.base.NewCaptchaID(),
				Key: "auth.captcha.invalid",
			}, http.StatusBadRequest)
			return
		}
	}

	uid, status := s.base.Signup(r.Context(), auth.Email, auth.Username, auth.Password, auth.Language, kilonova.PreferredTheme(auth.Theme), ip, &ua)
	if status != nil {
		if s.base.CaptchaEnabled() {
			errorData(w, struct {
				ID   string `json:"captcha_id"`
				Text string `json:"text"`
			}{
				ID:   s.base.NewCaptchaID(),
				Text: status.Error(),
			}, kilonova.ErrorCode(status))
		} else {
			statusError(w, status)
		}
		return
	}

	// Signup has no "keep me signed in" choice, so the first session is always a short one
	sid, err := s.base.CreateSession(r.Context(), uid, false)
	if err != nil {
		statusError(w, err)
		return
	}
	s.base.SetSessCookie(w, sid, false)
	returnData(w, sid)
}

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
