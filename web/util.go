package web

import (
	"net/http"

	"github.com/KiloProjects/Kilonova/common"
	"github.com/KiloProjects/Kilonova/internal/models"
	"github.com/KiloProjects/Kilonova/internal/util"
	"github.com/go-chi/chi"
)

// hydrateTemplate fills a templateData struct with generic stuff like Params, User and LoggedIn
func (rt *Web) hydrateTemplate(r *http.Request) templateData {
	return templateData{
		Params:   globParams(r),
		User:     util.UserFromContext(r),
		LoggedIn: util.IsRAuthed(r),
		Version:  common.Version,

		Problem: util.ProblemFromContext(r),
		Task:    util.TaskFromContext(r),
		Test:    util.TestFromContext(r),

		ProblemID: util.IDFromContext(r, util.PbID),
		TaskID:    util.IDFromContext(r, util.TaskID),
		TestID:    util.IDFromContext(r, util.TestID),

		// HACK: Move this somewhere else
		ProblemEditor: util.IsRProblemEditor(r),
		TaskEditor:    util.IsRTaskEditor(r),

		OGUrl: r.URL.RequestURI(),
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

type testDataType struct {
	In  string
	Out string
}

func (rt *Web) getFullTestData(test models.Test) testDataType {
	in, out, err := rt.dm.GetTest(test.ProblemID, test.VisibleID)
	if err != nil {
		in = []byte("err")
		out = []byte("err")
	}
	return testDataType{In: string(in), Out: string(out)}
}

func (rt *Web) getTestData(test models.Test) testDataType {
	t := rt.getFullTestData(test)
	if len(t.In) > 128*1024 { // 128KB
		t.In = "too long to show here"
	}
	if len(t.Out) > 128*1024 { // 128KB
		t.Out = "too long to show here"
	}
	return t
}
