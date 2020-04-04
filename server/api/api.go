package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
	"github.com/gorilla/securecookie"
	"github.com/jinzhu/gorm"
)

// API is the base
type API struct {
	ctx     context.Context
	db      *gorm.DB
	config  *models.Config
	session *securecookie.SecureCookie
}

// NewAPI declares a new API instance
func NewAPI(ctx context.Context, db *gorm.DB, config *models.Config) *API {
	session := securecookie.New([]byte(config.SecretKey), nil)
	session = session.SetSerializer(securecookie.JSONEncoder{})
	return &API{ctx, db, config, session}
}

// GetRouter is the magic behind the API
func (s *API) GetRouter() *chi.Mux {
	r := chi.NewRouter()

	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Pinged")
	})
	r.Mount("/auth", s.registerAuth())
	r.Mount("/problem", s.registerProblem())
	r.Mount("/motd", s.registerMOTD())
	r.Mount("/admin", s.registerAdmin())
	r.Mount("/tasks", s.registerTasks())
	r.Mount("/user", s.registerUser())
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "404", "error": "Not Found"})
	})
	return r
}

// GetSessionCookie reads and returns the data from the session cookie
func (s *API) GetSessionCookie(r *http.Request) *models.Session {
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

// SetSessionCookie sets the data to the session cookie
func (s *API) SetSessionCookie(w http.ResponseWriter, sess models.Session) error {
	encoded, err := s.session.Encode("kn-sessionid", sess)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:  "kn-sessionid",
		Value: encoded,
		Path:  "/",
	}
	http.SetCookie(w, cookie)
	return nil
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
