// Package domain — сущности и порты users-service.
package domain

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Ошибки домена: транспорт маппит их в error-envelope контракта.
var (
	// ErrNotFound — профиль не найден (404 not_found).
	ErrNotFound = errors.New("profile not found")
	// ErrForbidden — операция запрещена для текущего caller'а (403 forbidden).
	ErrForbidden = errors.New("forbidden")
	// ErrUnauthorized — нет/невалидный токен (401 unauthorized).
	ErrUnauthorized = errors.New("unauthorized")
)

// Profile — профиль пользователя. user_id = id в auth-service.
type Profile struct {
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
	Notes       *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// UpdateProfile — частичное обновление. nil = поле не передано (оставить как есть);
// пустая строка у avatar_url/notes = очистить (semantics согласованы с BFF).
type UpdateProfile struct {
	DisplayName *string
	AvatarURL   *string
	Notes       *string
}

// ProfileRepo — порт хранения профилей.
type ProfileRepo interface {
	// Create создаёт профиль; идемпотентно (повторный user_id — no-op).
	Create(ctx context.Context, p *Profile) error
	GetByID(ctx context.Context, userID uuid.UUID) (*Profile, error)
	// ListByIDs возвращает только найденные профили (порядок не гарантирован).
	ListByIDs(ctx context.Context, ids []uuid.UUID) ([]Profile, error)
	Update(ctx context.Context, userID uuid.UUID, upd UpdateProfile) (*Profile, error)
}

// OutboxEvent — событие для публикации (та же схема, что в auth-service).
type OutboxEvent struct {
	ID        uuid.UUID
	Type      string
	Payload   []byte
	CreatedAt time.Time
}

// OutboxRepo — порт transactional outbox.
type OutboxRepo interface {
	Insert(ctx context.Context, events []OutboxEvent) error
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, ids []uuid.UUID) error
}

// EventProcessedRepo — порт дедупликации входящих событий (at-least-once → exactly-once side effect).
type EventProcessedRepo interface {
	// MarkProcessed вставляет id события; false — событие уже было обработано.
	MarkProcessed(ctx context.Context, eventID uuid.UUID) (bool, error)
}
