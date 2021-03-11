package mdrenderer

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"

	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Renderer is a local markdown renderer. It does not depend on any external services but it does not support mathjax rendering or any extensions to the markdown standard I intend to make.
type LocalRenderer struct {
	md goldmark.Markdown
}

func (r *LocalRenderer) Render(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	err := r.md.Convert(src, &buf)
	return buf.Bytes(), err
}

func NewLocalRenderer() *LocalRenderer {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM, extension.Footnote, mathjax.MathJax),
		goldmark.WithParserOptions(parser.WithAutoHeadingID(), parser.WithAttribute()),
		goldmark.WithRendererOptions(html.WithHardWraps(), html.WithXHTML()),
	)
	return &LocalRenderer{md: md}
}

// ExternalRenderer talks to a HTTP server located at http://hostname:port/ through a multipart web request.
// It is intended to be the best way to parse markdown into HTML.
// It is slower than the LocalRenderer, but it also parses MathJax inputs, so it reduces overhead on the user side. It is also possible to cache the result so you don't even need to run it every time.
// TODO: Figure out how much slower
type ExternalRenderer struct {
	url string
}

func (r *ExternalRenderer) Render(src []byte) ([]byte, error) {

	var body bytes.Buffer
	// since the writer in multipart.Writer is a bytes.Buffer which never errors out at Write(), we can safely ignore all errors
	wr := multipart.NewWriter(&body)
	fw, _ := wr.CreateFormFile("md", "markdown.txt")
	fw.Write(src)
	wr.Close()
	req, err := http.NewRequest("POST", r.url, bytes.NewReader(body.Bytes()))
	if err != nil {
		log.Println(err)
		return nil, err
	}
	req.Header.Set("Content-Type", wr.FormDataContentType())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	ret, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
		return nil, err
	}
	return ret, resp.Body.Close()
}

func NewExternalRenderer(url string) *ExternalRenderer {
	return &ExternalRenderer{url}
}
