// Package pbedit holds the problem editor pages: general settings, access control, tests and subtasks.
package pbedit

import (
	"context"
	"embed"
	"encoding/json"
	"strings"

	"github.com/KiloProjects/kilonova"
	"github.com/KiloProjects/kilonova/web/ui/problems"
	"github.com/shopspring/decimal"
	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

//go:embed js/*.js
var scripts embed.FS

// script inlines js/name, replacing each __KEY__ token with the JSON encoding of vars[KEY].
// json.Marshal escapes <, > and &, so the values are safe inside <script>.
func script(name string, vars map[string]any) Node {
	b, err := scripts.ReadFile("js/" + name)
	if err != nil {
		panic(err)
	}
	pairs := make([]string, 0, 2*len(vars))
	for k, v := range vars {
		j, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		pairs = append(pairs, "__"+k+"__", string(j))
	}
	return Script(Raw(strings.NewReplacer(pairs...).Replace(string(b))))
}

// page is the shared editor frame: the topbar followed by the page holder.
func page(ctx context.Context, topbar *problems.TopbarParams, children ...Node) Node {
	return Group{
		problems.Topbar(ctx, topbar),
		Div(Class("page-holder"), Group(children)),
	}
}

func scoreStep(pb *kilonova.Problem) string {
	return decimal.NewFromInt(1).Shift(-pb.ScorePrecision).String()
}

// selectedClass reproduces the template's " list-group-selected " class suffix.
func selectedClass(base string, selected bool) string {
	if selected {
		return base + " list-group-selected"
	}
	return base
}
