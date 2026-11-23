package mq

// Юнит-тесты consumer'а: разбор конверта и применение user.created.

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/rosterra/services/users/internal/domain"
	"github.com/rosterra/services/users/internal/service"
)

func envelope(eventID, userID uuid.UUID, name *string) []byte {
	nameJSON := "null"
	if name != nil {
		nameJSON = `"` + *name + `"`
	}
	return []byte(`{
		"specversion": "1.0",
		"id": "` + eventID.String() + `",
		"type": "user.created",
		"source": "/rosterra/auth",
		"time": "2026-01-01T00:00:00Z",
		"data": {"id": "` + userID.String() + `", "email": "a@b.dev", "name": ` + nameJSON + `}
	}`)
}

func TestParseEvent(t *testing.T) {
	eventID, userID := uuid.New(), uuid.New()
	name := "Anna"

	evt, err := parseEvent(envelope(eventID, userID, &name))
	require.NoError(t, err)
	require.Equal(t, eventID.String(), evt.ID)
	require.Equal(t, "user.created", evt.Type)
}

func TestParseEventBroken(t *testing.T) {
	_, err := parseEvent([]byte(`{broken`))
	require.Error(t, err)

	_, err = parseEvent([]byte(`{"specversion":"1.0"}`))
	require.Error(t, err, "envelope без id/type должен отбиваться")
}

// applyEvent создаёт профиль из события; повторная доставка — no-op.
func TestApplyEventCreatesProfileIdempotent(t *testing.T) {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	profiles := newFakeProfileRepo()
	outbox := &fakeOutboxRepo{}
	processed := newFakeProcessedRepo()
	svc := service.New(profiles, outbox, processed, log)
	c := NewConsumer(svc, nil, log, nil)

	eventID, userID := uuid.New(), uuid.New()
	name := "FromEvent"

	require.NoError(t, c.applyEvent(context.Background(), mustParse(t, envelope(eventID, userID, &name))))

	stored, err := profiles.GetByID(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)
	require.False(t, stored.CreatedAt.IsZero())

	// Redelivery того же event_id — профиль не меняется.
	require.NoError(t, c.applyEvent(context.Background(), mustParse(t, envelope(eventID, userID, &name))))
	stored, err = profiles.GetByID(context.Background(), userID)
	require.NoError(t, err)
	require.Equal(t, name, stored.DisplayName)

	// Событие не-user.created игнорируется (Ack-путь валиден).
	blocked := mustParse(t, envelope(uuid.New(), uuid.New(), nil))
	blocked.Type = "user.blocked"
	require.NoError(t, c.applyEvent(context.Background(), blocked))
}

func mustParse(t *testing.T, raw []byte) cloudEvent {
	t.Helper()
	evt, err := parseEvent(raw)
	require.NoError(t, err)
	return evt
}

// компиляционная проверка: фейки совместимы с доменными портами.
var (
	_ domain.ProfileRepo        = (*fakeProfileRepo)(nil)
	_ domain.OutboxRepo         = (*fakeOutboxRepo)(nil)
	_ domain.EventProcessedRepo = (*fakeProcessedRepo)(nil)
)
