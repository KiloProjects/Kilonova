package scheduler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/KiloProjects/kilonova/eval"
	graderv1 "github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1"
	"github.com/KiloProjects/kilonova/eval/scheduler/proto/kilonova/grader/v1/graderv1connect"
)

var (
	errMissingToken = errors.New("missing or malformed bearer token")
	errUnknownToken = errors.New("unregistered client token")
)

// GraderServer exposes a local BoxManager + LanguageManager over ConnectRPC.
// It holds no platform credentials; every method just runs the in-process grader.
type GraderServer struct {
	graderv1connect.UnimplementedGraderServiceHandler

	sched eval.Box3Scheduler
	langs eval.LanguageManager
}

func NewGraderServer(sched eval.Box3Scheduler, langs eval.LanguageManager) *GraderServer {
	return &GraderServer{sched: sched, langs: langs}
}

// Handler returns the mount path and http.Handler, wrapped in the given auth
// interceptor. Mount it on the grader's TLS server.
func (s *GraderServer) Handler(reg *ClientRegistry) (string, http.Handler) {
	return graderv1connect.NewGraderServiceHandler(s, connect.WithInterceptors(newAuthInterceptor(reg)))
}

func (s *GraderServer) RunBox3(ctx context.Context, req *connect.Request[graderv1.RunBox3Request]) (*connect.Response[graderv1.RunBox3Response], error) {
	resp, err := s.sched.RunBox3(ctx, box3RequestFromProto(req.Msg.GetRequest()), req.Msg.GetMemQuota())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&graderv1.RunBox3Response{Response: box3ResponseToProto(resp)}), nil
}

func (s *GraderServer) RunMultibox3(ctx context.Context, req *connect.Request[graderv1.RunMultibox3Request]) (*connect.Response[graderv1.RunMultibox3Response], error) {
	resp, stats, err := s.sched.RunMultibox3(ctx, multibox3RequestFromProto(req.Msg.GetRequest()), req.Msg.GetManagerMemQuota(), req.Msg.GetIndividualMemQuota())
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&graderv1.RunMultibox3Response{
		ManagerResponse: box3ResponseToProto(resp),
		UserStats:       statsSliceToProto(stats),
	}), nil
}

func (s *GraderServer) Languages(ctx context.Context, req *connect.Request[graderv1.LanguagesRequest]) (*connect.Response[graderv1.LanguagesResponse], error) {
	return connect.NewResponse(&graderv1.LanguagesResponse{Versions: s.langs.LanguageVersions(ctx)}), nil
}

// --- auth ---

// ClientRegistry maps grader-minted bearer tokens to a named platform client.
// Priority is reserved (documented but unconsumed by this change).
type ClientRegistry struct {
	byToken map[string]ClientIdentity
}

type ClientIdentity struct {
	Name     string
	Priority string
}

func NewClientRegistry() *ClientRegistry {
	return &ClientRegistry{byToken: make(map[string]ClientIdentity)}
}

// Add registers a client token. An empty token is rejected so a misconfigured
// registry can never authenticate a caller that sends no token.
func (r *ClientRegistry) Add(token, name, priority string) error {
	if token == "" {
		return errors.New("client registry: empty token for client " + name)
	}
	r.byToken[token] = ClientIdentity{Name: name, Priority: priority}
	return nil
}

// authMiddleware wraps a plain http.Handler with the same bearer-token check as
// the RPC interceptor, so the scratch data plane shares one auth system.
func (r *ClientRegistry) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		token, ok := strings.CutPrefix(req.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			http.Error(w, "missing bearer token", http.StatusUnauthorized)
			return
		}
		if _, ok := r.byToken[token]; !ok {
			http.Error(w, "unregistered token", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, req)
	})
}

type clientNameKey struct{}

// ClientName returns the authenticated client name attached by the interceptor.
func ClientName(ctx context.Context) string {
	name, _ := ctx.Value(clientNameKey{}).(string)
	return name
}

func newAuthInterceptor(reg *ClientRegistry) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			token, ok := strings.CutPrefix(req.Header().Get("Authorization"), "Bearer ")
			if !ok || token == "" {
				return nil, connect.NewError(connect.CodeUnauthenticated, errMissingToken)
			}
			ident, ok := reg.byToken[token]
			if !ok {
				return nil, connect.NewError(connect.CodeUnauthenticated, errUnknownToken)
			}
			ctx = context.WithValue(ctx, clientNameKey{}, ident.Name)
			slog.DebugContext(ctx, "Authenticated grader RPC", slog.String("client", ident.Name), slog.String("procedure", req.Spec().Procedure))
			return next(ctx, req)
		}
	}
}
