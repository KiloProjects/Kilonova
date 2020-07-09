package web

import (
	"encoding/json"
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff liek Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request) templateData {
	user := common.UserFromContext(r)

	return templateData{
		Params:   rt.globParams(r),
		User:     &user,
		LoggedIn: user.ID != 0,
	}
}

func (rt *Web) globParams(r *http.Request) map[string]string {
	ctx := chi.RouteContext(r.Context())
	params := make(map[string]string)
	for i := 0; i < len(ctx.URLParams.Keys); i++ {
		params[ctx.URLParams.Keys[i]] = ctx.URLParams.Values[i]
	}
	return params
}

func (rt *Web) remarshal(in interface{}, out interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

type testDataType struct {
	In  string
	Out string
}

func (rt *Web) getTestData(test common.Test) testDataType {
	in, out, err := rt.dm.GetTest(test.ProblemID, test.VisibleID)
	if err != nil {
		in = []byte("err")
		out = []byte("err")
	}
	return testDataType{In: string(in), Out: string(out)}
}
