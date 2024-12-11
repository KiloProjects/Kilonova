package main

import (
	"bytes"
	"flag"
	"fmt"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/evanw/esbuild/pkg/api"
	"log"
	"os"
)

var (
	outFile = flag.String("o", "", "output file")
)

func main() {
	flag.Parse()
	formatter := chtml.New(chtml.WithClasses(true), chtml.TabWidth(4)) // Identical to mdrenderer.go
	var lightBuf, darkBuf bytes.Buffer
	if err := formatter.WriteCSS(&lightBuf, styles.Get("github")); err != nil {
		log.Println("Could not write `github` theme")
	}
	if err := formatter.WriteCSS(&darkBuf, styles.Get("github-dark")); err != nil {
		log.Println("Could not write `github-dark` theme")
	}
	css := fmt.Sprintf(".light {%s} .dark {%s}", lightBuf.String(), darkBuf.String())
	rez := api.Transform(css, api.TransformOptions{
		Loader: api.LoaderCSS,
		// MinifyWhitespace: true,
		Engines: []api.Engine{
			{Name: api.EngineChrome, Version: "100"},
			{Name: api.EngineFirefox, Version: "100"},
			{Name: api.EngineSafari, Version: "11"},
		},
	})

	if len(rez.Errors) > 0 {
		log.Fatalf("Found %d errors in chroma.css: %#v", len(rez.Errors), rez.Errors)
	}

	if err := os.WriteFile(*outFile, rez.Code, 0644); err != nil {
		log.Fatalf("Could not write `%s`", *outFile)
	}
}
