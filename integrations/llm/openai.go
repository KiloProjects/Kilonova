package llm

import (
	"context"
	"errors"

	"github.com/KiloProjects/kilonova/internal/config"
	"github.com/sashabaranov/go-openai"
)

var (
	Token = config.GenFlag("integrations.openai.token", "", "API Key for OpenAI access (used in translating statements)")

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
	client := openai.NewClient(Token.Value())
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
