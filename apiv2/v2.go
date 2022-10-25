package apiv2

import (
	"encoding/json"
	"net/http"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/sudoapi"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/schema"
)

var decoder *schema.Decoder

type API struct {
	base *sudoapi.BaseAPI

	sudoHandlers *sudoapi.WebHandler
}

// New declares a new API instance
func New(base *sudoapi.BaseAPI) *API {
	return &API{base, sudoapi.NewWebHandler(base)}
}

func (s *API) Handler() http.Handler {
	r := chi.NewRouter()

	return r
}

func init() {
	decoder = schema.NewDecoder()
	decoder.SetAliasTag("json")
}

func returnData(w http.ResponseWriter, retData any) {
	kilonova.StatusData(w, "success", retData, 200)
}

func errorData(w http.ResponseWriter, retData any, errCode int) {
	kilonova.StatusData(w, "error", retData, errCode)
}

func parseJsonBody[T any](r *http.Request, output *T) *kilonova.StatusError {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(output); err != nil {
		return kilonova.Statusf(400, "Invalid JSON input.")
	}
	return nil
}
