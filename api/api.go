package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
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

// Run is the magic behind the API
func (s *API) Run() {
	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Timeout(20 * time.Second))
	corsConfig := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	})
	r.Use(corsConfig.Handler)

	r.Route("/api/", func(r chi.Router) {

		r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
			log.Println("Pinged")
		})

		r.Mount("/auth", s.registerAuth())
		r.Mount("/problem", s.registerProblem())
		r.Mount("/motd", s.registerMOTD())
		r.Mount("/admin", s.registerAdmin())
		r.Mount("/tasks", s.registerTasks())
		r.Mount("/user", s.registerUser())
	})

	// graceful setup and shutdown
	server := &http.Server{Addr: ":3000", Handler: r}

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println(err)
		}
	}()

	// Waiting for SIGINT (pkill -2)
	<-s.ctx.Done()
	if err := server.Shutdown(s.ctx); err != nil {
		fmt.Println(err)
	}

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
