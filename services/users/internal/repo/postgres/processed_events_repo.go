package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// EventProcessedRepo — дедупликация входящих событий (таблица processed_events).
type EventProcessedRepo struct {
	pool *pgxpool.Pool
}

// NewEventProcessedRepo — fx-провайдер.
func NewEventProcessedRepo(pool *pgxpool.Pool) *EventProcessedRepo {
	return &EventProcessedRepo{pool: pool}
}

// MarkProcessed вставляет id события; возвращает false, если событие уже обработано.
func (r *EventProcessedRepo) MarkProcessed(ctx context.Context, eventID uuid.UUID) (bool, error) {
	tag, err := r.pool.Exec(ctx, `
		INSERT INTO processed_events (event_id) VALUES ($1)
		ON CONFLICT (event_id) DO NOTHING`,
		eventID)
	if err != nil {
		return false, fmt.Errorf("insert processed_event: %w", err)
	}
	return tag.RowsAffected() == 1, nil
}
