package mq

// Локальные фейки портов для тестов consumer'а (копия семантики из service/fakes_test.go).

import (
	"context"
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
		return nil
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

func (f *fakeProfileRepo) ListByIDs(_ context.Context, _ []uuid.UUID) ([]domain.Profile, error) {
	return nil, nil
}

func (f *fakeProfileRepo) Update(_ context.Context, _ uuid.UUID, _ domain.UpdateProfile) (*domain.Profile, error) {
	return nil, domain.ErrNotFound
}

type fakeOutboxRepo struct {
	mu sync.Mutex
}

func (f *fakeOutboxRepo) Insert(_ context.Context, _ []domain.OutboxEvent) error { return nil }
func (f *fakeOutboxRepo) FetchUnpublished(_ context.Context, _ int) ([]domain.OutboxEvent, error) {
	return nil, nil
}
func (f *fakeOutboxRepo) MarkPublished(_ context.Context, _ []uuid.UUID) error { return nil }

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
