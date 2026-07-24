package scheduler

import (
	"context"

	"connectrpc.com/connect"
	"github.com/KiloProjects/kilonova/eval"
	graderv1 "github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1"
	"github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1/graderv1connect"
)

var _ eval.Box3Scheduler = (*GraderClient)(nil)

// GraderClient is the platform-side eval.Box3Scheduler backed by a remote grader
// over ConnectRPC. It carries the grader-minted bearer token on every request.
type GraderClient struct {
	client graderv1connect.GraderServiceClient
}

func NewGraderClient(httpClient connect.HTTPClient, baseURL, token string, opts ...connect.ClientOption) *GraderClient {
	opts = append(opts, connect.WithInterceptors(bearerInterceptor(token)))
	return &GraderClient{client: graderv1connect.NewGraderServiceClient(httpClient, baseURL, opts...)}
}

func (c *GraderClient) RunBox3(ctx context.Context, req *eval.Box3Request, memQuota int64) (*eval.Box3Response, error) {
	resp, err := c.client.RunBox3(ctx, connect.NewRequest(&graderv1.RunBox3Request{
		Request:  box3RequestToProto(req),
		MemQuota: memQuota,
	}))
	if err != nil {
		return nil, err
	}
	return box3ResponseFromProto(resp.Msg.GetResponse()), nil
}

func (c *GraderClient) RunMultibox3(ctx context.Context, req *eval.Multibox3Request, managerMemQuota, individualMemQuota int64) (*eval.Box3Response, []*eval.RunStats, error) {
	resp, err := c.client.RunMultibox3(ctx, connect.NewRequest(&graderv1.RunMultibox3Request{
		Request:            multibox3RequestToProto(req),
		ManagerMemQuota:    managerMemQuota,
		IndividualMemQuota: individualMemQuota,
	}))
	if err != nil {
		return nil, nil, err
	}
	return box3ResponseFromProto(resp.Msg.GetManagerResponse()), statsSliceFromProto(resp.Msg.GetUserStats()), nil
}

// Close is a client-side no-op: the grader is shared across platform instances,
// so one platform shutting down must not drain the grader's boxes.
func (c *GraderClient) Close(ctx context.Context) error { return nil }

// languageVersions fetches the grader's supported language -> version map.
func (c *GraderClient) languageVersions(ctx context.Context) (map[string]string, error) {
	resp, err := c.client.Languages(ctx, connect.NewRequest(&graderv1.LanguagesRequest{}))
	if err != nil {
		return nil, err
	}
	return resp.Msg.GetVersions(), nil
}

func bearerInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if req.Spec().IsClient {
				req.Header().Set("Authorization", "Bearer "+token)
			}
			return next(ctx, req)
		}
	}
}
