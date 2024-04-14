package llm

import (
	"context"
	"errors"
	"net/http"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/sashabaranov/go-openai"
)

var (
	BaseURL      = config.GenFlag("integrations.openai.base_url", "", "Base URL for OpenAI API (`https://openrouter.ai/api/v1` can be used for OpenRouter)")
	Token        = config.GenFlag("integrations.openai.token", "", "API Key for OpenAI access (used in translating statements)")
	DefaultModel = config.GenFlag("integrations.openai.default_model", "gpt-4-turbo", "Default model for LLM translations")

	ErrUnauthed = errors.New("unauthenticated to OpenAI endpoint")
)

const translateRoSystemPrompt = `You will be given a competitive programming problem statement in markdown, written in the Romanian language, using GFM extensions and MathJax/LaTeX math between dollar signs ($ or $$). Another extension is the fact that image attachments are defined using a syntax similar to ~[name.png], with optional attributes named after the end with a vertical bar ('|'). 
You must translate the statement in the English language, while preserving mathematical values, variable names, general syntax, structure and format. You must also preserve the custom image format exactly as is. 
The "Cerință" task is always translated to "Task", "Date de intrare" is translated to "Input data", "Date de ieșire" is translated to "Output data". 
When separating large integers (especially in latex/mathjax math) in groups of 3 digits, do not add a comma. Instead, add a backslash followed and preceded by a single space character.
`

// TranslateStatement translates a statement from Romanian to English.
// English to Romanian and other language ordered pairs are still a TODO
func TranslateStatement(ctx context.Context, text string, model string) (string, error) {
	if len(Token.Value()) < 2 {
		return "", ErrUnauthed
	}
	if model == "" {
		model = DefaultModel.Value()
	}
	config := openai.DefaultConfig(Token.Value())
	config.HTTPClient = defaultClient
	if v := BaseURL.Value(); len(v) > 0 {
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
	Transport: &openRouterHeaderTransport{T: http.DefaultTransport},
}

type openRouterHeaderTransport struct {
	T http.RoundTripper
}

func (adt *openRouterHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	branding, ok := config.GetFlagVal[string]("frontend.navbar.branding")
	if !ok {
		branding = "Kilonova"
	}
	req.Header.Add("HTTP-Referer", branding)
	req.Header.Add("X-Title", "Kilonova")
	return adt.T.RoundTrip(req)
}
