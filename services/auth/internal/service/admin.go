package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/rosterra/services/auth/internal/domain"
)

// AdminResult — пользователь + факт публикации события (для логов).
type AdminResult struct {
	User       *domain.User
	RevokedAll bool // сессии отозваны (блокировка)
}

// ListUsers — admin-список с пагинацией и фильтрами (контракт /v1/admin/users).
func (s *Service) ListUsers(ctx context.Context, f domain.ListFilter) ([]domain.User, int, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}
	return s.users.List(ctx, f)
}

// GetUser — admin-карточка пользователя.
func (s *Service) GetUser(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	return s.users.GetByID(ctx, id)
}

// SetStatus меняет статус: блокировка отзывает все сессии; событие — в outbox той же транзакцией.
func (s *Service) SetStatus(ctx context.Context, id uuid.UUID, status domain.UserStatus, reason *string) (*AdminResult, error) {
	if status != domain.StatusActive && status != domain.StatusBlocked {
		return nil, validationError("status", "must be 'active' or 'blocked'")
	}

	eventType := domain.EventUserUnblocked
	payload := map[string]any{"user_id": id.String()}
	if status == domain.StatusBlocked {
		eventType = domain.EventUserBlocked
		if reason != nil && *reason != "" {
			payload["reason"] = *reason
		}
	}
	event, err := buildEvent(eventType, payload)
	if err != nil {
		return nil, err
	}

	user, err := s.users.UpdateStatus(ctx, id, status, &event)
	if err != nil {
		return nil, err
	}

	res := &AdminResult{User: user}
	if status == domain.StatusBlocked {
		if err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
			return nil, fmt.Errorf("revoke sessions: %w", err)
		}
		res.RevokedAll = true
	}
	s.log.Info("user status changed", "user_id", id, "status", string(status))
	return res, nil
}

// SetRoles заменяет роли пользователя; событие — в outbox той же транзакцией.
func (s *Service) SetRoles(ctx context.Context, id uuid.UUID, roles []domain.Role) (*domain.User, error) {
	if len(roles) == 0 {
		return nil, validationError("roles", "at least one role is required")
	}
	for _, r := range roles {
		if r != domain.RoleUser && r != domain.RoleAdmin {
			return nil, validationError("roles", "unknown role "+string(r))
		}
	}

	event, err := buildEvent(domain.EventRoleChanged, map[string]any{
		"user_id": id.String(),
		"roles":   roles,
	})
	if err != nil {
		return nil, err
	}
	user, err := s.users.SetRoles(ctx, id, roles, &event)
	if err != nil {
		return nil, err
	}
	s.log.Info("user roles changed", "user_id", id, "roles", fmt.Sprint(roles))
	return user, nil
}
