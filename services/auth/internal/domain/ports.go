package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// UserRepo — хранилище пользователей (PostgreSQL).
// Create / UpdateStatus / SetRoles выполняют изменения и событие outbox в одной транзакции.
type UserRepo interface {
	Create(ctx context.Context, u *User, passwordHash string, event *OutboxEvent) error
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status UserStatus, event *OutboxEvent) (*User, error)
	SetRoles(ctx context.Context, id uuid.UUID, roles []Role, event *OutboxEvent) (*User, error)
	List(ctx context.Context, f ListFilter) (items []User, total int, err error)
}

// SessionRepo — refresh-сессии (Redis), ключ — TokenHashHex.
type SessionRepo interface {
	// Create регистрирует новую сессию и добавляет её в семью и индекс пользователя.
	Create(ctx context.Context, rec RefreshSession, tokenHash string) error
	// Get возвращает сессию по хешу токена (ErrInvalidRefreshToken, если нет).
	Get(ctx context.Context, tokenHash string) (RefreshSession, error)
	// Rotate атомарно помечает старый токен заменённым и создаёт новый в той же семье.
	Rotate(ctx context.Context, oldHash string, newRec RefreshSession, newHash string) error
	// Revoke удаляет сессию (logout).
	Revoke(ctx context.Context, tokenHash string) error
	// RevokeFamily отзывает всю цепочку ротации (reuse detection).
	RevokeFamily(ctx context.Context, family string) error
	// RevokeAllForUser отзывает все семьи пользователя (блокировка).
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

// OutboxRepo — transactional outbox (PostgreSQL).
type OutboxRepo interface {
	FetchUnpublished(ctx context.Context, limit int) ([]OutboxEvent, error)
	MarkPublished(ctx context.Context, ids []uuid.UUID) error
}

// EventPublisher — публикация события в брокер (NATS JetStream).
type EventPublisher interface {
	Publish(ctx context.Context, evt OutboxEvent) error
}

// ListFilter — параметры admin-списка (контракт /v1/admin/users).
type ListFilter struct {
	Query   *string
	Status  *UserStatus
	Role    *Role
	Page    int // >= 1
	PerPage int // 1..100
}

// Темы и типы событий платформы. Схема: svc.<service>.<entity>.<action>.
const (
	EventUserCreated   = "user.created"
	EventUserBlocked   = "user.blocked"
	EventUserUnblocked = "user.unblocked"
	EventRoleChanged   = "user.role_changed"

	SubjectPrefix = "svc.auth."
)

// Subject возвращает тему брокера для типа события.
func (e OutboxEvent) Subject() string { return SubjectPrefix + e.Type }

// CE — CloudEvents 1.0 конверт (упрощённый).
type CE struct {
	SpecVersion string          `json:"specversion"`
	ID          string          `json:"id"`
	Type        string          `json:"type"`
	Source      string          `json:"source"`
	Time        time.Time       `json:"time"`
	Data        json.RawMessage `json:"data"`
}

// OutboxEvent — событие, публикуемое через transactional outbox.
type OutboxEvent struct {
	ID        uuid.UUID
	Type      string // EventUserCreated, ...
	Payload   json.RawMessage
	CreatedAt time.Time
}
