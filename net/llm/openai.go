// Package llm provides LLM-backed statement tooling behind a Provider
// interface. The composition root (cmd/kn) decides whether a provider exists;
// a nil Provider means the integration is not configured.
package llm

import (
	"context"
	"io"
	"net/http"

	"github.com/openai/openai-go/v3"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	_ "embed"
)

//go:embed prompts/translate.md
var translateRoSystemPrompt string

//go:embed prompts/transcribe.md
var transcribeStatement string

type Provider interface {
	// TranscribeStatement turns a PDF task statement into Markdown.
	TranscribeStatement(ctx context.Context, pdf io.Reader) (string, error)
	// TranslateStatement translates a Romanian statement to English.
	// Other language pairs are still a TODO.
	TranslateStatement(ctx context.Context, text string) (string, error)
}

type openAI struct {
	client      openai.Client
	textModel   string
	visionModel string
}

// NewOpenAI builds a Provider over the OpenAI Responses API. referer is sent
// as HTTP-Referer / X-Title for OpenRouter-style attribution.
func NewOpenAI(token, textModel, visionModel, referer string) Provider {
	client := openai.NewClient(
		option.WithAPIKey(token),
		option.WithHTTPClient(&http.Client{Transport: &refererTransport{referer: referer, T: otelhttp.NewTransport(http.DefaultTransport)}}),
	)
	return &openAI{client: client, textModel: textModel, visionModel: visionModel}
}

func (o *openAI) TranscribeStatement(ctx context.Context, pdfFile io.Reader) (string, error) {
	fResp, err := o.client.Files.New(ctx, openai.FileNewParams{
		File:    openai.File(pdfFile, "statement.pdf", "application/pdf"),
		Purpose: "user_data",
	})
	if err != nil {
		return "", err
	}

	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: o.visionModel,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam{
			{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleDeveloper,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: param.NewOpt(transcribeStatement),
					},
				},
			},
			{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfInputItemContentList: responses.ResponseInputMessageContentListParam{
							{
								OfInputText: &responses.ResponseInputTextParam{
									Text: "Transcribe this PDF task statement.",
								},
							},
							{
								OfInputFile: &responses.ResponseInputFileParam{
									FileID: param.NewOpt(fResp.ID),
								},
							},
						},
					},
				},
			},
		}},
	})
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

func (o *openAI) TranslateStatement(ctx context.Context, text string) (string, error) {
	resp, err := o.client.Responses.New(ctx, responses.ResponseNewParams{
		Model: o.textModel,
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: responses.ResponseInputParam{
			{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleDeveloper,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: param.NewOpt(translateRoSystemPrompt),
					},
				},
			},
			{
				OfMessage: &responses.EasyInputMessageParam{
					Role: responses.EasyInputMessageRoleUser,
					Content: responses.EasyInputMessageContentUnionParam{
						OfString: param.NewOpt(text),
					},
				},
			},
		}},
	})
	if err != nil {
		return "", err
	}
	return resp.OutputText(), nil
}

type refererTransport struct {
	referer string
	T       http.RoundTripper
}

func (t *refererTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("HTTP-Referer", t.referer)
	req.Header.Add("X-Title", "Kilonova")
	return t.T.RoundTrip(req)
}
