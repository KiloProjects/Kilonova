package web

import (
	"encoding/json"
	"fmt"
	"html/template"
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

func gradient(score, maxscore int) template.CSS {
	clamp := func(val, min, max int) int {
		if val < min {
			val = min
		}
		if val > max {
			val = max
		}
		return val
	}
	type color struct {
		red   int
		green int
		blue  int
	}
	score = clamp(score, 0, maxscore)
	percent := int(float64(score) / float64(maxscore) * 100)
	first := color{255, 130, 121}
	second := color{189, 255, 124}
	dr := -(first.red - second.red)
	dg := -(first.green - second.green)
	db := -(first.blue - second.blue)
	val := fmt.Sprintf("background-color: rgb(%d, %d, %d);",
		clamp(first.red+dr*percent/99, 0, 255),
		clamp(first.green+dg*percent/99, 0, 255),
		clamp(first.blue+db*percent/99, 0, 255),
	)
	return template.CSS(val)
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
