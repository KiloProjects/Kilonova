package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/AlexVasiluta/kilonova/datamanager"
	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
	"github.com/gorilla/securecookie"
	"github.com/jinzhu/gorm"
)

// RetData should be the way data will be returned
type RetData struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

// API is the base struct for the project's API
type API struct {
	ctx     context.Context
	db      *gorm.DB
	config  *models.Config
	session *securecookie.SecureCookie
	manager *datamanager.Manager
}

// NewAPI declares a new API instance
func NewAPI(ctx context.Context, db *gorm.DB, config *models.Config, manager *datamanager.Manager) *API {
	session := securecookie.New([]byte(config.SecretKey), nil)
	session = session.SetSerializer(securecookie.JSONEncoder{})
	return &API{ctx, db, config, session, manager}
}

// GetRouter is the magic behind the API
func (s *API) GetRouter() chi.Router {
	r := chi.NewRouter()

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "pong.")
	})
	r.Mount("/auth", s.RegisterAuthRoutes())
	r.Mount("/problem", s.RegisterProblemRoutes())
	r.Mount("/motd", s.RegisterMOTDRoutes())
	r.Mount("/admin", s.RegisterAdminRoutes())
	r.Mount("/tasks", s.RegisterTaskRoutes())
	r.Mount("/user", s.RegisterUserRoutes())
	return r
}

// ReturnData returns the json data to the user
func (s *API) ReturnData(w http.ResponseWriter, status string, returnData interface{}) {
	err := json.NewEncoder(w).Encode(RetData{
		Status: status,
		Data:   returnData,
	})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(RetData{
			Status: "error",
			Data:   err.Error(),
		})
	}
}

// ErrorData is like ReturnData but sets the corresponding error code in the header
func (s *API) ErrorData(w http.ResponseWriter, returnData interface{}, errCode int) {
	w.WriteHeader(errCode)
	s.ReturnData(w, "error", returnData)
}

// GetAuthToken returns the authentication token from a request
func (s *API) GetAuthToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	return ""
}

// GetSession reads and returns the data from the session cookie
func (s *API) GetSession(r *http.Request) *models.Session {
	authToken := s.GetAuthToken(r)
	if authToken != "" { // use Auth tokens by default
		var ret models.Session
		s.session.Decode("kn-sessionid", authToken, &ret)
		return &ret
	}
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		return nil
	}
	if cookie.Value == "" {
		return nil
	}
	var ret models.Session
	s.session.Decode(cookie.Name, cookie.Value, &ret)
	return &ret
}

// SetSession sets the data to the session cookie
func (s *API) SetSession(w http.ResponseWriter, sess models.Session) (string, error) {
	encoded, err := s.session.Encode("kn-sessionid", sess)
	if err != nil {
		return "", err
	}
	cookie := &http.Cookie{
		Name:     "kn-sessionid",
		Value:    encoded,
		Path:     "/",
		HttpOnly: false,
	}
	http.SetCookie(w, cookie)
	return encoded, nil
}

// RemoveSessionCookie clears the session cookie, effectively revoking it. When setting MaxAge to 0, the browser will also clear it out
func (s *API) RemoveSessionCookie(w http.ResponseWriter) {
	emptyCookie := &http.Cookie{
		Name:    "kn-sessionid",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),
	}
	http.SetCookie(w, emptyCookie)
}

func (s *API) getContextValue(r *http.Request, name string) interface{} {
	return r.Context().Value(models.KNContextType(name))
}
