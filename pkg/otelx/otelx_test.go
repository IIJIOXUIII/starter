package otelx

// Юнит-тесты наблюдаемости: пропагация W3C и спаны middleware.

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSetupNoopWithoutEndpoint(t *testing.T) {
	o, err := Setup(context.Background(), Config{ServiceName: "test"})
	require.NoError(t, err)
	require.NotNil(t, o)
	require.NoError(t, o.Shutdown(context.Background()))
}

func TestCloudEventTraceparentRoundtrip(t *testing.T) {
	// Реальный провайдер: создаём валидный span-контекст.
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "publish")
	traceparent := InjectCloudEvent(ctx)
	span.End()
	require.NotEmpty(t, traceparent, "traceparent должен записаться в CE-extension")

	// Потребитель продолжает трейс: span-контекст совпадает по TraceID.
	child := ContextFromCloudEvent(context.Background(), traceparent)
	require.True(t, trace.SpanContextFromContext(child).IsValid())
	require.Equal(t,
		trace.SpanContextFromContext(ctx).TraceID(),
		trace.SpanContextFromContext(child).TraceID(),
		"consumer продолжает тот же трейс",
	)

	// Пустой/битый traceparent → исходный контекст (новый корень).
	require.Equal(t, context.Background(), ContextFromCloudEvent(context.Background(), ""))
	require.Equal(t, context.Background(), ContextFromCloudEvent(context.Background(), "00-invalid"))
}

func TestTracingMiddlewareCreatesSpan(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	router := chi.NewRouter()
	router.Use(TracingMiddleware("test-svc"))
	router.Get("/v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/v1/ping", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Contains(t, spans[0].Name(), "/v1/ping")
	require.True(t, spans[0].SpanContext().IsValid())
}

func TestTracingMiddlewarePropagatesIncomingTrace(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	prev := otel.GetTracerProvider()
	otel.SetTracerProvider(tp)
	defer otel.SetTracerProvider(prev)

	// "Клиент" начинает трейс и шлёт traceparent.
	clientRec := tracetest.NewSpanRecorder()
	clientTP := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(clientRec))
	clientTracer := clientTP.Tracer("client")
	ctx, span := clientTracer.Start(context.Background(), "outgoing")
	header := http.Header{}
	Propagator.Inject(ctx, propagation.HeaderCarrier(header))
	span.End()

	router := chi.NewRouter()
	router.Use(TracingMiddleware("test-svc"))
	router.Get("/v1/x", func(http.ResponseWriter, *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "/v1/x", nil)
	req.Header = header
	router.ServeHTTP(httptest.NewRecorder(), req)

	serverSpans := recorder.Ended()
	require.Len(t, serverSpans, 1)
	require.Equal(t,
		trace.SpanContextFromContext(ctx).TraceID(),
		serverSpans[0].SpanContext().TraceID(),
		"серверный спан продолжается из traceparent клиента",
	)
}
