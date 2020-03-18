package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/AlexVasiluta/kilonova/models"
	"github.com/davecgh/go-spew/spew"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/securecookie"
	"github.com/jinzhu/gorm"
)

var (
	session *securecookie.SecureCookie
)

// API is the base
type API struct {
	ctx    context.Context
	db     *gorm.DB
	config *models.Config
}

// NewAPI declares a new API instance
func NewAPI(ctx context.Context, db *gorm.DB, config *models.Config) *API {
	return &API{
		ctx, db, config,
	}
}

// Run is the magic behind the API
func (s *API) Run() {
	session = securecookie.New([]byte(s.config.SecretKey), nil)
	session = session.SetSerializer(securecookie.JSONEncoder{})

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

		r.Get("/getResults", func(w http.ResponseWriter, r *http.Request) {
			var allTasks []models.Task
			s.db.Find(&allTasks)
			json.NewEncoder(w).Encode(allTasks)
		})

		r.Get("/getUserByName/{name}", func(w http.ResponseWriter, r *http.Request) {
			var user models.User
			s.db.First(&user, "name = ?", chi.URLParam(r, "name"))
			json.NewEncoder(w).Encode(user)
		})

		r.Get("/getUsers", func(w http.ResponseWriter, r *http.Request) {
			var users []models.User
			s.db.Find(&users)
			json.NewEncoder(w).Encode(users)
		})

		r.Post("/submit", func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			spew.Dump(r.PostForm)
			w.Header().Set("location", "http://localhost:3000/")
			w.WriteHeader(http.StatusMovedPermanently)
		})

		r.Mount("/auth", s.registerAuth())

		r.Mount("/problem", s.registerProblem())

		r.Mount("/motd", s.registerMOTD())

		r.Mount("/admin", s.registerAdmin())
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

func getViewFunc(templates *template.Template, name string, args interface{}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := templates.ExecuteTemplate(w, name, args)
		if err != nil {
			fmt.Fprintln(w, err)
		}
	}
}

// HandleTests fills a models.Test array with the tests from an archive
func HandleTests(r *http.Request) ([]models.Test, error) {
	return nil, nil
}

// GetSessionCookie reads and returns the data from the session cookie
func GetSessionCookie(w http.ResponseWriter, r *http.Request) models.Session {
	cookie, err := r.Cookie("kn-sessionid")
	if err != nil {
		fmt.Println(err)
		return models.Session{}
	}
	var ret models.Session
	session.Decode(cookie.Name, cookie.Value, &ret)
	return models.Session{}
}

// SetSessionCookie sets the data to the session cookie
func SetSessionCookie(w http.ResponseWriter, sess models.Session) error {
	encoded, err := session.Encode("kn-sessionid", sess)
	if err != nil {
		return err
	}
	cookie := &http.Cookie{
		Name:  "kn-sessionid",
		Value: encoded,
	}
	http.SetCookie(w, cookie)
	return nil
}
