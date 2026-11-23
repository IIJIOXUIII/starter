package otelx

// HTTP-middleware: серверные спаны и метрики Prometheus.
// Route-лейбл берётся из chi RoutePattern — защищает от cardinality-взрыва.

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// MetricsMiddleware — http_requests_total и http_request_duration_seconds
// с лейблами route из chi-контекста.
func MetricsMiddleware(serviceName string) func(http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "HTTP requests processed.",
	}, []string{"service", "method", "route", "status"})
	duration := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request duration.",
		Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5},
	}, []string{"service", "method", "route"})

	prometheus.MustRegister(requests, duration)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)

			route := routePattern(r)
			method := normalizedMethod(r.Method)
			requests.WithLabelValues(serviceName, method, route, strconv.Itoa(rec.status)).Inc()
			duration.WithLabelValues(serviceName, method, route).Observe(time.Since(start).Seconds())
		})
	}
}

// TracingMiddleware — серверный спан на запрос; контекст извлекается из
// W3C-заголовков (traceparent от BFF/браузера).
func TracingMiddleware(serviceName string) func(http.Handler) http.Handler {
	tracer := otel.Tracer(serviceName)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := Propagator.Extract(r.Context(), propagationHeaders(r.Header))
			// Имя спана уточняется после dispatch (chi вычисляет шаблон маршрута
			// в процессе выполнения цепочки).
			spanCtx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", r.Method, r.URL.Path),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					attribute.String("http.request.method", normalizedMethod(r.Method)),
					attribute.String("url.path", r.URL.Path),
					attribute.String("server.address", r.Host),
				),
			)
			defer span.End()

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r.WithContext(spanCtx))

			if routeCtx := chi.RouteContext(spanCtx); routeCtx != nil {
				if pattern := routeCtx.RoutePattern(); pattern != "" {
					span.SetName(fmt.Sprintf("%s %s", r.Method, pattern))
				}
			}
			span.SetAttributes(attribute.Int("http.response.status_code", rec.status))
			if rec.status >= 500 {
				span.SetStatus(codes.Error, fmt.Sprintf("http %d", rec.status))
			}
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// routePattern — chi-шаблон маршрута ("/v1/admin/users/{id}"), fallback на путь.
func routePattern(r *http.Request) string {
	if ctx := chi.RouteContext(r.Context()); ctx != nil {
		if pattern := ctx.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}

// normalizedMethod — HEAD считается семантикой GET (рекомендация OTel).
func normalizedMethod(method string) string {
	if method == http.MethodHead {
		return http.MethodGet
	}
	return method
}

// propagationHeaders адаптирует http.Header к TextMapCarrier.
type propagationHeaders http.Header

func (h propagationHeaders) Get(key string) string { return http.Header(h).Get(key) }
func (h propagationHeaders) Set(key, value string) { http.Header(h).Set(key, value) }
func (h propagationHeaders) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}
