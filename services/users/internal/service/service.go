// Package service — бизнес-логика users-service: профили.
package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/rosterra/services/users/internal/domain"
)

// Лимиты полей профиля (дублируют контракт users.yaml).
const (
	maxDisplayName = 100
	maxAvatarURL   = 2048
	maxNotes       = 2000
	maxBatchIDs    = 100
)

// ValidateError — некорректные данные (400 validation_failed + details).
type ValidateError struct {
	Fields []FieldError
}

// FieldError — ошибка конкретного поля.
type FieldError struct {
	Field   string
	Message string
}

func (e *ValidateError) Error() string {
	return "validation failed"
}

func validationError(field, message string) *ValidateError {
	return &ValidateError{Fields: []FieldError{{Field: field, Message: message}}}
}

// eventSource — источник событий users в CloudEvents-конверте.
const eventSource = "/rosterra/users"

// EventTypeProfileUpdated — профиль изменён.
const EventTypeProfileUpdated = "profile.updated"

// Service — сервис профилей.
type Service struct {
	profiles  domain.ProfileRepo
	outbox    domain.OutboxRepo
	processer domain.EventProcessedRepo
	log       *slog.Logger
	now       func() time.Time
}

// New собирает сервис.
func New(profiles domain.ProfileRepo, outbox domain.OutboxRepo, processer domain.EventProcessedRepo, log *slog.Logger) *Service {
	return &Service{profiles: profiles, outbox: outbox, processer: processer, log: log, now: time.Now}
}

// CreateProfile создаёт профиль. Идемпотентно: повторный вызов — no-op
// (первый источник — событие user.created, повторные — gRPC/NATS redelivery).
func (s *Service) CreateProfile(ctx context.Context, userID uuid.UUID, displayName, avatarURL string) (*domain.Profile, error) {
	if displayName = strings.TrimSpace(displayName); len(displayName) > maxDisplayName {
		return nil, validationError("display_name", fmt.Sprintf("max %d characters", maxDisplayName))
	}
	if len(avatarURL) > maxAvatarURL {
		return nil, validationError("avatar_url", fmt.Sprintf("max %d characters", maxAvatarURL))
	}

	p := &domain.Profile{UserID: userID, DisplayName: displayName}
	if avatarURL != "" {
		avatar := avatarURL
		p.AvatarURL = &avatar
	}
	if err := s.profiles.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create profile: %w", err)
	}
	return s.profiles.GetByID(ctx, userID)
}

// GetProfile возвращает профиль.
func (s *Service) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.Profile, error) {
	if errors.Is(ctx.Err(), context.Canceled) {
		return nil, ctx.Err()
	}
	p, err := s.profiles.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return p, nil
}

// ListProfiles — batch по id (для агрегации admin-списка в BFF).
func (s *Service) ListProfiles(ctx context.Context, ids []uuid.UUID) ([]domain.Profile, error) {
	if len(ids) == 0 {
		return []domain.Profile{}, nil
	}
	if len(ids) > maxBatchIDs {
		return nil, validationError("ids", fmt.Sprintf("max %d ids per request", maxBatchIDs))
	}
	return s.profiles.ListByIDs(ctx, ids)
}

// UpdateProfile — частичное обновление + событие profile.updated в outbox
// (релиз публикует его в NATS; сам UPDATE — один атомарный запрос репозитория).
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, upd domain.UpdateProfile) (*domain.Profile, error) {
	if upd.DisplayName != nil {
		v := strings.TrimSpace(*upd.DisplayName)
		if len(v) > maxDisplayName {
			return nil, validationError("display_name", fmt.Sprintf("max %d characters", maxDisplayName))
		}
		upd.DisplayName = &v
	}
	if upd.AvatarURL != nil && *upd.AvatarURL != "" && len(*upd.AvatarURL) > maxAvatarURL {
		return nil, validationError("avatar_url", fmt.Sprintf("max %d characters", maxAvatarURL))
	}
	if upd.Notes != nil && len(*upd.Notes) > maxNotes {
		return nil, validationError("notes", fmt.Sprintf("max %d characters", maxNotes))
	}

	p, err := s.profiles.Update(ctx, userID, upd)
	if err != nil {
		return nil, err
	}

	// profile.updated — в outbox; публикует релей.
	if err := s.emitUpdated(ctx, p); err != nil {
		// Событие критично для консистентности копий, но профиль уже обновлён:
		// логируем и продолжаем (релей повторит; outbox хранится в той же БД).
		s.log.Error("enqueue profile.updated", "user_id", userID, "err", err)
	}
	return p, nil
}

// HandleUserCreated — применение события user.created (consumer NATS).
// Идемпотентность: dedup по event.id + ON CONFLICT в репозитории.
func (s *Service) HandleUserCreated(ctx context.Context, eventID uuid.UUID, userID uuid.UUID, name *string) error {
	first, err := s.processer.MarkProcessed(ctx, eventID)
	if err != nil {
		return fmt.Errorf("mark processed: %w", err)
	}
	if !first {
		s.log.Info("event already processed, skip", "event_id", eventID, "user_id", userID)
		return nil
	}

	displayName := ""
	if name != nil {
		displayName = *name
	}
	if _, err := s.CreateProfile(ctx, userID, displayName, ""); err != nil {
		return fmt.Errorf("create profile from event: %w", err)
	}
	s.log.Info("profile created from event", "user_id", userID)
	return nil
}

// emitUpdated кладёт profile.updated в outbox.
func (s *Service) emitUpdated(ctx context.Context, p *domain.Profile) error {
	payload := map[string]any{
		"user_id":      p.UserID.String(),
		"display_name": p.DisplayName,
		"updated_at":   p.UpdatedAt,
	}
	evt := domain.OutboxEvent{
		ID:        uuid.New(),
		Type:      EventTypeProfileUpdated,
		CreatedAt: s.now().UTC(),
	}
	raw, err := json.Marshal(map[string]any{
		"specversion": "1.0",
		"id":          evt.ID.String(),
		"type":        evt.Type,
		"source":      eventSource,
		"time":        evt.CreatedAt,
		"data":        payload,
	})
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}
	evt.Payload = raw
	return s.outbox.Insert(ctx, []domain.OutboxEvent{evt})
}
