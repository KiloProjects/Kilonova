package api

import "github.com/go-chi/chi"

func (s *API) registerAdmin() chi.Router {
	r := chi.NewRouter()
	return r
}
