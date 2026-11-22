package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/rosterra/services/auth/internal/domain"
)

// OutboxRepo — PostgreSQL-реализация domain.OutboxRepo.
type OutboxRepo struct {
	pool *pgxpool.Pool
}

// NewOutboxRepo конструирует репозиторий outbox.
func NewOutboxRepo(pool *pgxpool.Pool) *OutboxRepo {
	return &OutboxRepo{pool: pool}
}

// FetchUnpublished возвращает порцию неопубликованных событий (старые вперёд).
func (r *OutboxRepo) FetchUnpublished(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, type, payload, created_at
		FROM outbox
		WHERE published_at IS NULL
		ORDER BY created_at
		LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("select outbox: %w", err)
	}
	defer rows.Close()

	events := make([]domain.OutboxEvent, 0, limit)
	for rows.Next() {
		var e domain.OutboxEvent
		if err := rows.Scan(&e.ID, &e.Type, &e.Payload, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

// MarkPublished помечает события опубликованными (idempotent-релей).
func (r *OutboxRepo) MarkPublished(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE outbox SET published_at = $1 WHERE id = ANY($2)`,
		time.Now(), ids)
	if err != nil {
		return fmt.Errorf("mark published: %w", err)
	}
	// Часть строк могла быть отмечена параллельным релеем — не ошибка.
	return nil
}
