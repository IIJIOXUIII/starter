// Package otelx — общая наблюдаемость сервисов платформы:
//
//   - трассировка OpenTelemetry (OTLP HTTP exporter, W3C traceparent);
//   - HTTP-middleware: серверные спаны + метрики Prometheus (route из chi);
//   - gRPC unary-интерцепторы (сервер/клиент);
//   - пропагация контекста через NATS-сообщения (headers);
//   - сквозная корреляция через CloudEvents-extension "traceparent"
//     (событие auth → consumer users оказываются в одном трейсе).
//
// Инициализация ленивая: пустой OTEL_ENDPOINT → noop-провайдер
// (метрики при этом работают всегда).
package otelx

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
)

// Config — параметры трассировки.
type Config struct {
	// ServiceName — имя сервиса в ресурсе (отображается в Jaeger).
	ServiceName string
	// OTLPEndpoint — базовый URL OTLP HTTP (например, http://localhost:4318).
	// Пусто → noop-провайдер (спаны не создаются).
	OTLPEndpoint string
	// SampleRatio — доля трейсов (parentbased_traceidratio). 0 → 1.0.
	SampleRatio float64
}

// OTel — провайдеры наблюдаемости + shutdown.
type OTel struct {
	tracerProvider *sdktrace.TracerProvider
}

// Propagator — единый формат пропагации платформы (W3C).
var Propagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// Setup инициализирует глобальный TracerProvider и пропагатор.
// При пустом endpoint трассировка выключена (noop), но Propagator
// всё равно устанавливается: заголовки должны передаваться сквозь сервисы.
func Setup(ctx context.Context, cfg Config) (*OTel, error) {
	otel.SetTracerProvider(trace.NewNoopTracerProvider())
	otel.SetTextMapPropagator(Propagator)

	if cfg.OTLPEndpoint == "" {
		return &OTel{}, nil
	}

	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(stripScheme(cfg.OTLPEndpoint)),
		otlptracehttp.WithInsecure(), // dev-стек без TLS; в prod — терминируется на гейте
	)
	if err != nil {
		return nil, fmt.Errorf("otlp exporter: %w", err)
	}

	ratio := cfg.SampleRatio
	if ratio <= 0 {
		ratio = 1.0
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter, sdktrace.WithBatchTimeout(2*time.Second)),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName(cfg.ServiceName),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(ratio))),
	)
	otel.SetTracerProvider(tp)
	return &OTel{tracerProvider: tp}, nil
}

// Shutdown дожидается отправки незавершённых батчей (graceful).
func (o *OTel) Shutdown(ctx context.Context) error {
	if o == nil || o.tracerProvider == nil {
		return nil
	}
	return o.tracerProvider.Shutdown(ctx)
}

// stripScheme — OTLP HTTP exporter принимает host:port.
func stripScheme(endpoint string) string {
	for _, prefix := range []string{"http://", "https://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}
