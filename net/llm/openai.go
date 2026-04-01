package llm

import (
	"cmp"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/KiloProjects/kilonova/sudoapi/flags"
	"github.com/openai/openai-go/v3/option"
	"github.com/openai/openai-go/v3/packages/param"
	"github.com/openai/openai-go/v3/responses"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/openai/openai-go/v3"

	_ "embed"
)

var (
	ErrUnauthed = errors.New("unauthenticated to OpenAI endpoint")
)

//go:embed prompts/translate.md
var translateRoSystemPrompt string

//go:embed prompts/transcribe.md
var transcribeStatement string

func TranscribeStatement(ctx context.Context, pdfFile io.Reader, model string) (string, error) {
	if len(flags.OpenAIToken.Value()) < 2 {
		return "", ErrUnauthed
	}
	if model == "" {
		model = flags.OpenAIVisionModel.Value()
	}
	client := openai.NewClient(
		option.WithAPIKey(flags.OpenAIToken.Value()),
		//option.WithBaseURL(flags.OpenAIBaseURL.Value()),
		option.WithHTTPClient(defaultClient),
	)

	fResp, err := client.Files.New(ctx, openai.FileNewParams{
		File:    openai.File(pdfFile, "statement.pdf", "application/pdf"),
		Purpose: "user_data",
	})
	if err != nil {
		return "", err
	}

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: model,
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

// TranslateStatement translates a statement from Romanian to English.
// English to Romanian and other language ordered pairs are still a TODO
func TranslateStatement(ctx context.Context, text string, model string) (string, error) {
	if len(flags.OpenAIToken.Value()) < 2 {
		return "", ErrUnauthed
	}
	if model == "" {
		model = flags.OpenAIDefaultModel.Value()
	}
	client := openai.NewClient(
		option.WithAPIKey(flags.OpenAIToken.Value()),
		//option.WithBaseURL(flags.OpenAIBaseURL.Value()),
		option.WithHTTPClient(defaultClient),
	)

	resp, err := client.Responses.New(ctx, responses.ResponseNewParams{
		Model: model,
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
