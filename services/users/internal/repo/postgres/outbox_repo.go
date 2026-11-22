package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rosterra/services/users/internal/domain"
)

// OutboxRepo — domain.OutboxRepo на PostgreSQL (та же схема, что в auth-service).
type OutboxRepo struct {
	pool *pgxpool.Pool
}

// NewOutboxRepo — fx-провайдер.
func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

// Insert сохраняет события.
func (r *OutboxRepo) Insert(ctx context.Context, events []domain.OutboxEvent) error {
	for _, evt := range events {
		if _, err := r.pool.Exec(ctx,
			`INSERT INTO users_outbox (id, type, payload) VALUES ($1, $2, $3)`,
			evt.ID, evt.Type, evt.Payload); err != nil {
			return fmt.Errorf("insert outbox event: %w", err)
		}
	}
	return nil
}

// FetchUnpublished — порция неопубликованных событий (по created_at).
func (r *OutboxRepo) FetchUnpublished(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, payload, created_at
		FROM users_outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0, limit)
	for rows.Next() {
		var evt domain.OutboxEvent
		if err := rows.Scan(&evt.ID, &evt.Type, &evt.Payload, &evt.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows outbox: %w", err)
	}
	return events, nil
}

// MarkPublished помечает события опубликованными.
func (r *OutboxRepo) MarkPublished(ctx context.Context, ids []uuid.UUID) error {
	if _, err := r.pool.Exec(ctx,
		`UPDATE users_outbox SET published_at = now() WHERE id = ANY($1)`, ids); err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	return nil
}
