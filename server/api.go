package server

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/datamanager"
	"github.com/KiloProjects/Kilonova/kndb"
	"github.com/go-chi/chi"
)

// API is the base struct for the project's API
type API struct {
	ctx     context.Context
	db      *kndb.DB
	config  *common.Config
	manager datamanager.Manager
	logger  *log.Logger
}

// NewAPI declares a new API instance
func NewAPI(ctx context.Context, db *kndb.DB, config *common.Config, manager datamanager.Manager) *API {

	var logger *log.Logger
	if config.Debug {
		logger = log.New(os.Stdout, "[API]", log.LstdFlags)
	}
	return &API{ctx, db, config, manager, logger}
}

// GetRouter is the magic behind the API
func (s *API) GetRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(s.SetupSession)

	// /ping
	r.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		s.ReturnData(w, "success", "pong")
	})
	r.Mount("/auth", s.RegisterAuthRoutes())
	r.Mount("/problem", s.RegisterProblemRoutes())
	r.Mount("/admin", s.RegisterAdminRoutes())
	r.Mount("/tasks", s.RegisterTaskRoutes())
	r.Mount("/user", s.RegisterUserRoutes())
	return r
}

// ReturnData returns the json data to the user
func (s *API) ReturnData(w http.ResponseWriter, status string, returnData interface{}) {
	s.StatusData(w, status, returnData, 200)
}

// StatusData calls API.ReturnData but also sets a status code
func (s *API) StatusData(w http.ResponseWriter, status string, returnData interface{}, statusCode int) {
	w.WriteHeader(statusCode)
	err := json.NewEncoder(w).Encode(common.RetData{
		Status: status,
		Data:   returnData,
	})
	if err != nil {
		if err != nil {
			s.errlog("Couldn't send return data: %v", err)
		}
	}
}

// ErrorData calls API.ReturnData but sets the corresponding error code in the header
// NOTE: this is shorthand for API.StatusData(w, "error", returnData, errCode)
func (s *API) ErrorData(w http.ResponseWriter, returnData interface{}, errCode int) {
	s.StatusData(w, "error", returnData, errCode)
}

func (s *API) getContextValue(r *http.Request, name string) interface{} {
	return r.Context().Value(common.KNContextType(name))
}

func (s *API) log(format string, data ...interface{}) {
	s.logger.Printf(format, data...)
}

func (s *API) errlog(format string, data ...interface{}) {
	s.logger.Printf("[ERROR] "+format, data...)
}
