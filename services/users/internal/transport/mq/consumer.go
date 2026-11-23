package mq

import (
	"context"
	"encoding/json"
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
	"github.com/rosterra/services/users/internal/service"
)

// Параметры consumer'а событий auth.
const (
	// AuthUserCreatedSubject — subject события user.created из auth (svc.auth.>).
	AuthUserCreatedSubject = "svc.auth.user.created"
	// AuthStreamName — JetStream-стрим, который создаёт outbox-релей auth-service.
	AuthStreamName = "ROSTERRA_AUTH"
	// DurableName — durable-консьюмер: состояние в JetStream, рестарты безопасны.
	DurableName = "users_profiles_creator"
)

// cloudEvent — конверт CloudEvents 1.0 (как публикует auth outbox-релей).
type cloudEvent struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	Time        time.Time       `json:"time"`
	Data        json.RawMessage `json:"data"`
	// Traceparent — CE-extension: трейс публикации (auth) продолжается здесь.
	Traceparent string `json:"traceparent,omitempty"`
}

// userCreatedData — data события user.created (publicUser из auth-service).
type userCreatedData struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  *string   `json:"name"`
}

// maxDeliverAttempts — poison-message защита: после N redelivery — Term.
const maxDeliverAttempts = 5

// Consumer — фоновый потребитель user.created: создаёт профили новым пользователям.
type Consumer struct {
	svc    *service.Service
	js     nats.JetStreamContext
	log    *slog.Logger
	tracer trace.Tracer
	stop   chan struct{}
	done   chan struct{}
}

// NewConsumer собирает consumer.
// otel — фиксация порядка инициализации (см. http.NewServer).
func NewConsumer(svc *service.Service, js nats.JetStreamContext, log *slog.Logger, _ *otelx.OTel) *Consumer {
	return &Consumer{svc: svc, js: js, log: log, tracer: otel.Tracer("rosterra/users/consumer"), stop: make(chan struct{}), done: make(chan struct{})}
}

// Run подписывается (durable pull-consumer) и обрабатывает события до остановки.
func (c *Consumer) Run(ctx context.Context) {
	defer close(c.done)

	// Стрим auth-событий создаёт релей auth-service — ждём его появления.
	for {
		if _, err := c.js.StreamInfo(AuthStreamName); err == nil {
			break
		} else {
			c.log.Info("waiting for auth stream", "stream", AuthStreamName)
		}
		select {
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		case <-time.After(2 * time.Second):
		}
	}

	sub, err := c.js.PullSubscribe(AuthUserCreatedSubject, DurableName,
		nats.ManualAck(), nats.DeliverAll())
	if err != nil {
		c.log.Error("pull subscribe", "err", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stop:
			return
		default:
		}

		msgs, err := sub.Fetch(10, nats.MaxWait(2*time.Second))
		if errors.Is(err, nats.ErrTimeout) {
			continue
		}
		if err != nil {
			c.log.Error("fetch events", "err", err)
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
			continue
		}

		for _, msg := range msgs {
			c.handleMsg(ctx, msg)
		}
	}
}

// Stop останавливает consumer.
func (c *Consumer) Stop() {
	close(c.stop)
	select {
	case <-c.done:
	case <-time.After(5 * time.Second):
	}
}

// handleMsg разбирает событие и создаёт профиль; идемпотентность — dedup по event id.
func (c *Consumer) handleMsg(ctx context.Context, msg *nats.Msg) {
	evt, err := parseEvent(msg.Data)
	if err != nil {
		c.log.Error("parse event — terminating poison message", "err", err, "subject", msg.Subject)
		_ = msg.Term()
		return
	}

	meta, _ := msg.Metadata()
	if meta != nil && meta.NumDelivered > maxDeliverAttempts {
		c.log.Error("event failed too many times — terminating",
			"event_id", evt.ID, "attempts", meta.NumDelivered, "err_last", "see logs")
		_ = msg.Term()
		return
	}

	if err := c.applyEvent(ctx, evt); err != nil {
		c.log.Error("apply event — nak for redelivery", "event_id", evt.ID, "attempt", deliveryOf(meta), "err", err)
		_ = msg.Nak()
		return
	}
	if err := msg.Ack(); err != nil {
		c.log.Warn("ack event", "event_id", evt.ID, "err", err)
	}
}

// applyEvent парсит data и применяет создание профиля.
// Трейс: если в конверте есть traceparent — consumer продолжает трейс auth.
func (c *Consumer) applyEvent(ctx context.Context, evt cloudEvent) error {
	ctx = otelx.ContextFromCloudEvent(ctx, evt.Traceparent)
	ctx, span := c.tracer.Start(ctx, fmt.Sprintf("users.event.consume %s", evt.Type),
		trace.WithSpanKind(trace.SpanKindConsumer),
		trace.WithAttributes(
			attribute.String("messaging.system", "nats"),
			attribute.String("messaging.destination.name", AuthUserCreatedSubject),
			attribute.String("messaging.message.id", evt.ID),
		),
	)
	defer span.End()

	if evt.Type != "user.created" {
		c.log.Warn("unexpected event type, skipping", "type", evt.Type)
		return nil
	}
	eventID, err := uuid.Parse(evt.ID)
	if err != nil {
		span.SetStatus(codes.Error, "parse event id")
		return fmt.Errorf("parse event id: %w", err)
	}
	var data userCreatedData
	if err := json.Unmarshal(evt.Data, &data); err != nil {
		span.SetStatus(codes.Error, "unmarshal data")
		return fmt.Errorf("unmarshal user.created data: %w", err)
	}
	if err := c.svc.HandleUserCreated(ctx, eventID, data.ID, data.Name); err != nil {
		span.SetStatus(codes.Error, err.Error())
		return fmt.Errorf("handle user.created: %w", err)
	}
	return nil
}

// parseEvent разбирает конверт события.
func parseEvent(raw []byte) (cloudEvent, error) {
	var evt cloudEvent
	if err := json.Unmarshal(raw, &evt); err != nil {
		return cloudEvent{}, fmt.Errorf("unmarshal cloud event: %w", err)
	}
	if evt.ID == "" || evt.Type == "" {
		return cloudEvent{}, errors.New("event envelope missing id/type")
	}
	return evt, nil
}

// deliveryOf — число попыток доставки (0, если метаданные недоступны).
func deliveryOf(meta *nats.MsgMetadata) uint64 {
	if meta == nil {
		return 0
	}
	return meta.NumDelivered
}
