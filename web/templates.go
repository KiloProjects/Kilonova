package web

import (
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"strings"
)

// templates.go has a few functions for parsing templates

const suffix = ".templ"

// parseAllTemplates parses all templates in the specified fs.
func parseAllTemplates(t *template.Template, root fs.FS) (*template.Template, error) {
	err := fs.WalkDir(root, ".", func(fpath string, info os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(fpath, suffix) {
			return nil
		}

		absPath := strings.TrimSuffix(fpath, suffix)
		absPath = strings.TrimPrefix(absPath, "/")

		var tmpl *template.Template
		if tmpl == nil {
			tmpl = template.New(absPath)
		}
		if absPath == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(absPath)
		}

		dat, err := fs.ReadFile(root, fpath)
		if err != nil {
			panic(fmt.Sprintf("READ ERROR FOR FILE %s: %v\n", fpath, err))
		}

		if _, err = tmpl.Parse(string(dat)); err != nil {
			panic(fmt.Sprintf("PARSE ERROR FOR FILE %s: %v\n", fpath, err))
		}

		return nil
	})
	return t, err
}
