package service

// Фейки репозиториев для юнит-тестов (без БД).
// Семантика зеркалит SQL-репозиторий: Create — ON CONFLICT DO NOTHING,
// Update — nil = не менять, "" = очистить.

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/rosterra/services/users/internal/domain"
)

type fakeProfileRepo struct {
	mu       sync.Mutex
	profiles map[uuid.UUID]domain.Profile
}

func newFakeProfileRepo() *fakeProfileRepo {
	return &fakeProfileRepo{profiles: make(map[uuid.UUID]domain.Profile)}
}

func (f *fakeProfileRepo) Create(_ context.Context, p *domain.Profile) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.profiles[p.UserID]; ok {
		return nil // ON CONFLICT DO NOTHING
	}
	p.CreatedAt = time.Now().UTC()
	p.UpdatedAt = p.CreatedAt
	f.profiles[p.UserID] = *p
	return nil
}

func (f *fakeProfileRepo) GetByID(_ context.Context, userID uuid.UUID) (*domain.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.profiles[userID]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &p, nil
}

func (f *fakeProfileRepo) ListByIDs(_ context.Context, ids []uuid.UUID) ([]domain.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	items := make([]domain.Profile, 0, len(ids))
	for _, id := range ids {
		if p, ok := f.profiles[id]; ok {
			items = append(items, p)
		}
	}
	return items, nil
}

// Update применяет частичное обновление и является upsert: если профиль ещё
// не создан (лаг события user.created), правка создаёт его из переданных полей.
// nil = не менять, "" = очистить.
func (f *fakeProfileRepo) Update(_ context.Context, userID uuid.UUID, upd domain.UpdateProfile) (*domain.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	p, ok := f.profiles[userID]
	if !ok {
		p = domain.Profile{UserID: userID, CreatedAt: time.Now().UTC()}
	}
	if upd.DisplayName != nil {
		p.DisplayName = *upd.DisplayName
	}
	if upd.AvatarURL != nil {
		if *upd.AvatarURL == "" {
			p.AvatarURL = nil
		} else {
			v := *upd.AvatarURL
			p.AvatarURL = &v
		}
	}
	if upd.Notes != nil {
		if *upd.Notes == "" {
			p.Notes = nil
		} else {
			v := *upd.Notes
			p.Notes = &v
		}
	}
	p.UpdatedAt = time.Now().UTC()
	f.profiles[userID] = p
	return &p, nil
}

type fakeOutboxRepo struct {
	mu     sync.Mutex
	events []domain.OutboxEvent
}

func (f *fakeOutboxRepo) Insert(_ context.Context, events []domain.OutboxEvent) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, events...)
	return nil
}

func (f *fakeOutboxRepo) FetchUnpublished(_ context.Context, _ int) ([]domain.OutboxEvent, error) {
	return nil, nil
}

func (f *fakeOutboxRepo) MarkPublished(_ context.Context, _ []uuid.UUID) error { return nil }

func (f *fakeOutboxRepo) published() []domain.OutboxEvent {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]domain.OutboxEvent(nil), f.events...)
}

type fakeProcessedRepo struct {
	mu   sync.Mutex
	seen map[uuid.UUID]bool
}

func newFakeProcessedRepo() *fakeProcessedRepo {
	return &fakeProcessedRepo{seen: make(map[uuid.UUID]bool)}
}

func (f *fakeProcessedRepo) MarkProcessed(_ context.Context, eventID uuid.UUID) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.seen[eventID] {
		return false, nil
	}
	f.seen[eventID] = true
	return true, nil
}

// longString — строка-заглушка заданной длины.
func longString(n int) string {
	return strings.Repeat("a", n)
}
