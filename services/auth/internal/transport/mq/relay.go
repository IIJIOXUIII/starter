// Package mq — публикация событий outbox в NATS JetStream (at-least-once).
//
// Релей опрашивает таблицу outbox, публикует события в subjects svc.auth.>
// и помечает их опубликованными. Идемпотентность потребителей — по event.id.
package mq

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"github.com/rosterra/services/auth/internal/domain"
)

// Параметры JetStream-стрима auth-событий.
const (
	StreamName     = "ROSTERRA_AUTH"
	StreamSubjects = "svc.auth.>"
)

// Relay — фоновый публикатор outbox-событий.
type Relay struct {
	ob       domain.OutboxRepo
	js       nats.JetStreamContext
	log      *slog.Logger
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewRelay создаёт релей (интервал опроса — обычно 1s).
func NewRelay(ob domain.OutboxRepo, js nats.JetStreamContext, interval time.Duration, log *slog.Logger) *Relay {
	return &Relay{
		ob:       ob,
		js:       js,
		log:      log,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// EnsureStream идемпотентно создаёт JetStream-стрим для auth-событий.
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

// Stop останавливает релей и ждёт завершения (graceful shutdown).
func (r *Relay) Stop() {
	close(r.stop)
	select {
	case <-r.done:
	case <-time.After(5 * time.Second):
	}
}

// drain публикует порцию событий и помечает их опубликованными.
// Порядок сохраняется: при ошибке публикации продолжаем со следующего цикла.
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
		if _, err := r.js.PublishMsg(&nats.Msg{
			Subject: evt.Subject(),
			Data:    evt.Payload,
			Header:  nats.Header{"Nats-Msg-Id": []string{evt.ID.String()}}, // дедупликация в JetStream
		}, nats.AckWait(5*time.Second)); err != nil {
			r.log.Error("publish event", "id", evt.ID, "type", evt.Type, "err", err)
			break
		}
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
