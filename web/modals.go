package web

import (
	"fmt"
	"github.com/KiloProjects/kilonova"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/KiloProjects/kilonova/internal/util"
)

func (rt *Web) updateProblemSources() http.HandlerFunc {
	templ := rt.parseModal(nil, "modals/htmx/problem_sources.html")
	return func(w http.ResponseWriter, r *http.Request) {
		r = rt.buildPblistCache(r, []int{util.ProblemList(r).ID})

		var numUpdated int
		for i, id := range util.ProblemList(r).ProblemIDs() {
			source := fmt.Sprintf(r.FormValue("new_format"), i+1)
			if err := rt.base.UpdateProblem(r.Context(), id, kilonova.ProblemUpdate{SourceCredits: &source}, util.UserBrief(r)); err != nil {
				slog.WarnContext(r.Context(), "Could not update problem", slog.Any("err", err))
			} else {
				numUpdated++
			}
		}

		if numUpdated == 0 {
			htmxErrorToast(w, r, "No problem source was updated.")
		} else if numUpdated == len(util.ProblemList(r).ProblemIDs()) {
			htmxSuccessToast(w, r, "Updated problem sources.")
		} else {
			htmxInfoToast(w, r, "Not all problem sources could be updated.")
		}

		if isHTMXRequest(r) {
			rt.runModal(w, r, templ, "change_sources", util.ProblemList(r))
			return
		}

		http.Redirect(w, r, "/problem_lists/"+strconv.Itoa(util.ProblemList(r).ID), http.StatusTemporaryRedirect)
	}
}
