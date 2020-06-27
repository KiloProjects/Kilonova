package web

import (
	"encoding/json"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff liek Params, User and LoggedIn
func hydrateTemplate(r *http.Request) templateData {
	return templateData{
		Params:   globParams(r),
		User:     common.UserFromContext(r),
		LoggedIn: common.UserFromContext(r) != common.User{},
	}
}

func globParams(r *http.Request) map[string]string {
	ctx := chi.RouteContext(r.Context())
	params := make(map[string]string)
	for i := 0; i < len(ctx.URLParams.Keys); i++ {
		params[ctx.URLParams.Keys[i]] = ctx.URLParams.Values[i]
	}
	return params
}

func remarshal(in interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}
