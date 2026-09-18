package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/KiloProjects/kilonova/sudoapi/mdrenderer"
	chtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

var (
	outFile = flag.String("o", "", "output file")
)

func main() {
	flag.Parse()
	formatter := chtml.New(mdrenderer.HighlightFormatOptions()...)
	var lightBuf, darkBuf bytes.Buffer
	if err := formatter.WriteCSS(&lightBuf, styles.Get("github")); err != nil {
		log.Println("Could not write `github` theme")
	}
	if err := formatter.WriteCSS(&darkBuf, styles.Get("github-dark")); err != nil {
		log.Println("Could not write `github-dark` theme")
	}
	// Emitted with CSS nesting. Vite lowers it for whatever build.target says, so
	// this only has to be valid CSS, not portable CSS.
	css := fmt.Sprintf(".light {%s}\n.dark {%s}\n", lightBuf.String(), darkBuf.String())

	if err := os.WriteFile(*outFile, []byte(css), 0644); err != nil {
		log.Fatalf("Could not write `%s`", *outFile)
	}
}
