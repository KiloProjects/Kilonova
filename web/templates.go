package web

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"strings"

	"github.com/markbates/pkger"
)

// templates.go has a few functions for parsing templates using pkger
// because I can't specify a custom filesystem for ParseGlob or ParseFiles

const suffix = ".templ"
const root = "/web/templ/"

// parseAllTemplates parses all templates in the specified root (remember that in pkger, the root directory is the one with go.mod)
// note that the root will be stripped
func parseAllTemplates(t *template.Template, root string) (*template.Template, error) {

	pkger.Walk(root, func(fpath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(fpath, suffix) {
			return nil
		}

		absPath := strings.SplitN(fpath, ":", 2)[1]
		absPath = strings.TrimPrefix(absPath, root)
		absPath = strings.TrimSuffix(absPath, suffix)

		var tmpl *template.Template
		if tmpl == nil {
			tmpl = template.New(absPath)
		}
		if absPath == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(absPath)
		}

		file, err := pkger.Open(fpath)
		if err != nil {
			panic(fmt.Sprintf("OPEN ERROR FOR FILE %s: %v\n", fpath, err))
		}

		dat, err := io.ReadAll(file)
		if err != nil {
			panic(fmt.Sprintf("READ ERROR FOR FILE %s: %v\n", fpath, err))
		}

		if _, err = tmpl.Parse(string(dat)); err != nil {
			panic(fmt.Sprintf("PARSE ERROR FOR FILE %s: %v\n", fpath, err))
		}

		return nil
	})
	return t, nil
}
