// Package mq — публикация outbox-событий users-service в NATS JetStream
// (subjects svc.users.>) и потребление событий auth (svc.auth.user.created).
package mq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/rosterra/platform/pkg/otelx"
	"github.com/rosterra/services/users/internal/domain"
)

// Параметры JetStream-стрима users-событий.
const (
	StreamName     = "ROSTERRA_USERS"
	StreamSubjects = "svc.users.>"
)

// Relay — фоновый публикатор outbox-событий (зеркало auth-релея).
type Relay struct {
	ob       domain.OutboxRepo
	js       nats.JetStreamContext
	log      *slog.Logger
	interval time.Duration
	tracer   trace.Tracer
	stop     chan struct{}
	done     chan struct{}
}

// NewRelay создаёт релей (интервал опроса — обычно 1s).
// otel — фиксация порядка инициализации (см. http.NewServer).
func NewRelay(ob domain.OutboxRepo, js nats.JetStreamContext, interval time.Duration, log *slog.Logger, _ *otelx.OTel) *Relay {
	return &Relay{
		ob:       ob,
		js:       js,
		log:      log,
		interval: interval,
		tracer:   otel.Tracer("rosterra/users/outbox-relay"),
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// EnsureStream идемпотентно создаёт JetStream-стрим для users-событий.
func (r *Relay) EnsureStream() error {
	_, err := r.js.AddStream(&nats.StreamConfig{
		Name:      StreamName,
		Subjects:  []string{StreamSubjects},
		Retention: nats.LimitsPolicy,
	})
	if err != nil && !errors.Is(err, nats.ErrStreamNameAlreadyInUse) {
		return err
	}
	return nil
}

// Run запускает цикл релея; блокируется до Stop или отмены контекста.
func (r *Relay) Run(ctx context.Context) {
	defer close(r.done)
	if err := r.EnsureStream(); err != nil {
		r.log.Error("ensure jetstream stream", "err", err)
		return
	}

	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		if err := r.drain(ctx); err != nil {
			r.log.Error("relay drain", "err", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-r.stop:
			return
		case <-ticker.C:
		}
	}
}

// Stop останавливает релей и ждёт завершения.
func (r *Relay) Stop() {
	close(r.stop)
	select {
	case <-r.done:
	case <-time.After(5 * time.Second):
	}
}

// drain публикует порцию событий и помечает их опубликованными.
func (r *Relay) drain(ctx context.Context) error {
	events, err := r.ob.FetchUnpublished(ctx, 100)
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}

	published := make([]uuid.UUID, 0, len(events))
	for _, evt := range events {
		ctx, span := r.tracer.Start(ctx, fmt.Sprintf("outbox.publish %s", evt.Type),
			trace.WithSpanKind(trace.SpanKindProducer),
			trace.WithAttributes(
				attribute.String("messaging.system", "nats"),
				attribute.String("messaging.destination.name", "svc.users."+evt.Type),
				attribute.String("messaging.message.id", evt.ID.String()),
			),
		)
		msg := &nats.Msg{
			Subject: "svc.users." + evt.Type,
			Data:    evt.Payload,
			Header:  nats.Header{"Nats-Msg-Id": []string{evt.ID.String()}},
		}
		otelx.InjectNATS(ctx, msg)
		if _, err := r.js.PublishMsg(msg, nats.AckWait(5*time.Second)); err != nil {
			span.SetStatus(codes.Error, err.Error())
			span.End()
			r.log.Error("publish event", "id", evt.ID, "type", evt.Type, "err", err)
			break
		}
		span.End()
		published = append(published, evt.ID)
	}

	if len(published) > 0 {
		if err := r.ob.MarkPublished(ctx, published); err != nil {
			return err
		}
		r.log.Debug("events published", "count", len(published))
	}
	return nil
}
