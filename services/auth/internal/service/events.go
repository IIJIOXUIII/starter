package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/rosterra/services/auth/internal/domain"
)

// eventSource — источник событий в CloudEvents-конверте.
const eventSource = "/rosterra/auth"

// buildEvent собирает OutboxEvent с CloudEvents 1.0 payload.
// Публикация в NATS выполняется релеем, запись — в той же транзакции, что и изменение.
func buildEvent(eventType string, data any) (domain.OutboxEvent, error) {
	dataRaw, err := json.Marshal(data)
	if err != nil {
		return domain.OutboxEvent{}, fmt.Errorf("marshal event data: %w", err)
	}
	envelope := domain.CE{
		SpecVersion: "1.0",
		ID:          uuid.NewString(),
		Type:        eventType,
		Source:      eventSource,
		Time:        time.Now().UTC(),
		Data:        dataRaw,
	}
	raw, err := json.Marshal(envelope)
	if err != nil {
		return domain.OutboxEvent{}, fmt.Errorf("marshal event envelope: %w", err)
	}
	return domain.OutboxEvent{
		ID:        uuid.New(),
		Type:      eventType,
		Payload:   raw,
		CreatedAt: time.Now().UTC(),
	}, nil
}

// publicUser — публичное представление пользователя для событий (без password_hash).
type publicUser struct {
	ID            uuid.UUID `json:"id"`
	Email         string    `json:"email"`
	Name          *string   `json:"name"`
	Status        string    `json:"status"`
	Roles         []string  `json:"roles"`
	EmailVerified bool      `json:"email_verified"`
	CreatedAt     time.Time `json:"created_at"`
}

func toPublicUser(u *domain.User) publicUser {
	roles := make([]string, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, string(r))
	}
	return publicUser{
		ID:            u.ID,
		Email:         u.Email,
		Name:          u.Name,
		Status:        string(u.Status),
		Roles:         roles,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
	}
}
