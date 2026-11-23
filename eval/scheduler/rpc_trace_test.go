package scheduler

import (
	"context"
	"testing"

	"github.com/KiloProjects/kilonova/eval"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

// TestTraceContextCrossesWire proves the grader continues the platform's
// trace. That needs the client to inject traceparent and the server to extract
// it; drop either half and the grader's span lands in a trace of its own.
func TestTraceContextCrossesWire(t *testing.T) {
	rec := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(rec))
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	url := startTestGrader(t, &recordingSched{}, nil, "tok")
	ctx, root := tp.Tracer("test").Start(context.Background(), "eval")

	_, err := NewGraderClient(nil, url, "tok").RunBox3(ctx, &eval.Box3Request{Command: []string{"/bin/true"}}, 0)
	require.NoError(t, err)

	var got []string
	for _, s := range rec.Ended() {
		if s.SpanKind() == trace.SpanKindServer && s.SpanContext().TraceID() == root.SpanContext().TraceID() {
			got = append(got, s.Name())
		}
	}
	require.Equal(t, []string{"POST /run/box3"}, got)
}
