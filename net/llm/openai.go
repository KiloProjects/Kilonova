package llm

import (
	"cmp"
	"context"
	"errors"
	"net/http"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/sashabaranov/go-openai"
)

var (
	ErrUnauthed = errors.New("unauthenticated to OpenAI endpoint")
)

const translateRoSystemPrompt = `You will be given a competitive programming problem statement in markdown, written in the Romanian language, using GFM extensions and MathJax/LaTeX math between dollar signs ($ or $$). Another extension is the fact that image attachments are defined using a syntax similar to ~[name.png], with optional attributes named after the end with a vertical bar ('|').
You must translate the statement in the English language, while preserving mathematical values, variable names, general syntax, structure and format. You must also preserve the custom image format exactly as is.

The "Cerință" task is always translated to "Task", "Date de intrare" is translated to "Input data", "Date de ieșire" is translated to "Output data", "subsecvență" is translated to "subarray", "subșir" is translated to "subsequence", "Restricții și precizări" to "Constraints and clarifications".

In addition, in the "Date de intrare" and "Date de ieșire" sections, if you see expressions such as "Pe prima linie", "Pe a doua linie" etc., you want to use the verb "contain" to describe the data we need to read or print. In the "Date de ieșire" sections, you can also use the verb "print" to describe the data we need to print.

When separating large integers (especially in latex/mathjax math) in groups of 3 digits, do not add a comma. Instead, add a backslash followed and preceded by a single space character.

After you are done with translating, please double check the statement and fix potential grammar and/or syntax errors according to the rules of English language. Do not attempt to solve the problem.
`

// TranslateStatement translates a statement from Romanian to English.
// English to Romanian and other language ordered pairs are still a TODO
func TranslateStatement(ctx context.Context, text string, model string) (string, error) {
	if len(flags.OpenAIToken.Value()) < 2 {
		return "", ErrUnauthed
	}
	if model == "" {
		model = flags.OpenAIDefaultModel.Value()
	}
	config := openai.DefaultConfig(flags.OpenAIToken.Value())
	config.HTTPClient = defaultClient
	if v := flags.OpenAIBaseURL.Value(); len(v) > 0 {
		config.BaseURL = v
	}
	client := openai.NewClientWithConfig(config)
	// openai.NewClientWithConfig(ope)
	resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: model,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: translateRoSystemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: text},
		},
	})
	if err != nil {
		return "", err
	}
	return resp.Choices[0].Message.Content, nil
}

var defaultClient = &http.Client{
	Transport: &openRouterHeaderTransport{T: otelhttp.NewTransport(http.DefaultTransport)},
}

type openRouterHeaderTransport struct {
	T http.RoundTripper
}

func (adt *openRouterHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	branding := cmp.Or(flags.NavbarBranding.Value(), "Kilonova")
	req.Header.Add("HTTP-Referer", branding)
	req.Header.Add("X-Title", "Kilonova")
	return adt.T.RoundTrip(req)
}
