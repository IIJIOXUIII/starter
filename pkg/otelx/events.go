package otelx

// Пропагация трассировки через NATS-сообщения и CloudEvents-конверт.
//
// Схема сквозного трейса платформы:
//
//	HTTP register (auth) → outbox-событие с traceparent-расширением (в payload CE)
//	→ NATS relay публикует + инжектит в headers → consumer (users) извлекает
//	→ спаны consumer'а и последующих операций оказываются в том же трейсе.
//
// W3C traceparent хранится и в headers NATS (живой путь publish→consume),
// и в CloudEvents-extension (устойчивость: payload переживает хранение в outbox).

import (
	"context"

	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// TraceparentHeader — имя CloudEvents-extension и NATS-заголовка.
const TraceparentHeader = "traceparent"

// natsCarrier адаптирует nats.Header к TextMapCarrier.
type natsCarrier nats.Header

func (h natsCarrier) Get(key string) string {
	values := nats.Header(h).Get(key)
	return values
}

func (h natsCarrier) Set(key, value string) {
	nats.Header(h).Set(key, value)
}

func (h natsCarrier) Keys() []string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	return keys
}

// InjectNATS переносит текущий трейс-контекст в headers сообщения.
func InjectNATS(ctx context.Context, msg *nats.Msg) {
	if msg.Header == nil {
		msg.Header = nats.Header{}
	}
	Propagator.Inject(ctx, natsCarrier(msg.Header))
}

// ContextFromNATS извлекает трейс-контекст из headers сообщения.
func ContextFromNATS(ctx context.Context, msg *nats.Msg) context.Context {
	return Propagator.Extract(ctx, natsCarrier(msg.Header))
}

// InjectCloudEvent возвращает traceparent текущего контекста для записи
// в CloudEvents-конверт (extension-атрибута "traceparent").
func InjectCloudEvent(ctx context.Context) string {
	carrier := propagation.MapCarrier{}
	Propagator.Inject(ctx, carrier)
	return carrier.Get(TraceparentHeader)
}

// ContextFromCloudEvent продолжает трейс из CloudEvents-extension.
// Пустая extension → контекст не меняется (новый корень).
func ContextFromCloudEvent(ctx context.Context, traceparent string) context.Context {
	if traceparent == "" {
		return ctx
	}
	carrier := propagation.MapCarrier{TraceparentHeader: traceparent}
	// Валидность проверит сам пропагатор: при битом значении вернёт исходный ctx.
	sc := trace.SpanContextFromContext(Propagator.Extract(ctx, carrier))
	if !sc.IsValid() {
		return ctx
	}
	return Propagator.Extract(ctx, carrier)
}
