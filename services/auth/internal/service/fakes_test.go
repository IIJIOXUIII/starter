package service_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/alexedwards/argon2id"
	"github.com/google/uuid"
	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/services/auth/internal/domain"
	"github.com/rosterra/services/auth/internal/service"
)

// ---- fakes ----

type fakeUserRepo struct {
	byEmail map[string]*domain.User
	byID    map[uuid.UUID]*domain.User
	events  []string // типы записанных outbox-событий
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{byEmail: map[string]*domain.User{}, byID: map[uuid.UUID]*domain.User{}}
}

func (f *fakeUserRepo) Create(_ context.Context, u *domain.User, hash string, event *domain.OutboxEvent) error {
	key := strings.ToLower(u.Email)
	if _, ok := f.byEmail[key]; ok {
		return fmt.Errorf("duplicate email: %w", domain.ErrEmailTaken)
	}
	u.PasswordHash = hash
	f.byEmail[key] = u
	f.byID[u.ID] = u
	if event != nil {
		f.events = append(f.events, event.Type)
	}
	return nil
}

func (f *fakeUserRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	if u, ok := f.byEmail[strings.ToLower(email)]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	if u, ok := f.byID[id]; ok {
		return u, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUserRepo) UpdateStatus(_ context.Context, id uuid.UUID, status domain.UserStatus, event *domain.OutboxEvent) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u.Status = status
	if event != nil {
		f.events = append(f.events, event.Type)
	}
	return u, nil
}

func (f *fakeUserRepo) SetRoles(_ context.Context, id uuid.UUID, roles []domain.Role, event *domain.OutboxEvent) (*domain.User, error) {
	u, ok := f.byID[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	u.Roles = roles
	if event != nil {
		f.events = append(f.events, event.Type)
	}
	return u, nil
}

func (f *fakeUserRepo) List(context.Context, domain.ListFilter) ([]domain.User, int, error) {
	return nil, 0, nil
}

type fakeSessionRepo struct {
	sessions map[string]domain.RefreshSession // key: hash
	families map[string]map[string]bool       // family -> hashes
	userFams map[uuid.UUID]map[string]bool
}

func newFakeSessionRepo() *fakeSessionRepo {
	return &fakeSessionRepo{
		sessions: map[string]domain.RefreshSession{},
		families: map[string]map[string]bool{},
		userFams: map[uuid.UUID]map[string]bool{},
	}
}

func (f *fakeSessionRepo) Create(_ context.Context, rec domain.RefreshSession, hash string) error {
	f.sessions[hash] = rec
	f.addFamily(rec.Family, hash)
	f.addUser(rec.UserID, rec.Family)
	return nil
}

func (f *fakeSessionRepo) addFamily(fam, hash string) {
	if f.families[fam] == nil {
		f.families[fam] = map[string]bool{}
	}
	f.families[fam][hash] = true
}

func (f *fakeSessionRepo) addUser(uid uuid.UUID, fam string) {
	if f.userFams[uid] == nil {
		f.userFams[uid] = map[string]bool{}
	}
	f.userFams[uid][fam] = true
}

func (f *fakeSessionRepo) Get(_ context.Context, hash string) (domain.RefreshSession, error) {
	rec, ok := f.sessions[hash]
	if !ok {
		return domain.RefreshSession{}, domain.ErrInvalidRefreshToken
	}
	return rec, nil
}

func (f *fakeSessionRepo) Rotate(_ context.Context, oldHash string, newRec domain.RefreshSession, newHash string) error {
	rec, ok := f.sessions[oldHash]
	if !ok {
		return domain.ErrInvalidRefreshToken
	}
	if rec.ReplacedBy != "" {
		f.revokeFamily(rec.Family)
		return domain.ErrSessionRevoked
	}
	rec.ReplacedBy = newHash
	f.sessions[oldHash] = rec
	f.sessions[newHash] = newRec
	f.addFamily(newRec.Family, newHash)
	return nil
}

func (f *fakeSessionRepo) Revoke(_ context.Context, hash string) error {
	rec, ok := f.sessions[hash]
	if !ok {
		return nil
	}
	delete(f.sessions, hash)
	if fam, ok := f.families[rec.Family]; ok {
		delete(fam, hash)
		if len(fam) == 0 {
			delete(f.families, rec.Family)
			if u, ok := f.userFams[rec.UserID]; ok {
				delete(u, rec.Family)
			}
		}
	}
	return nil
}

func (f *fakeSessionRepo) revokeFamily(fam string) {
	for h := range f.families[fam] {
		delete(f.sessions, h)
	}
	delete(f.families, fam)
}

func (f *fakeSessionRepo) RevokeFamily(_ context.Context, family string) error {
	f.revokeFamily(family)
	return nil
}

func (f *fakeSessionRepo) RevokeAllForUser(_ context.Context, uid uuid.UUID) error {
	for fam := range f.userFams[uid] {
		f.revokeFamily(fam)
	}
	delete(f.userFams, uid)
	return nil
}

// ---- helpers ----

const testTTL = time.Hour

func newTestService(t *testing.T) (*service.Service, *fakeUserRepo, *fakeSessionRepo) {
	t.Helper()
	users := newFakeUserRepo()
	sessions := newFakeSessionRepo()
	// 64 hex-символа = 32 байта seed Ed25519.
	const seed = "a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6a8b0c2d4e6f8a0b2c4d6e8f0a2b4"
	issuer, _, err := jwtx.NewIssuer(seed, "rosterra-auth", "rosterra", 15*time.Minute)
	if err != nil {
		t.Fatalf("issuer: %v", err)
	}
	// Ослабленные параметры Argon2id — чтобы тесты не шли минуты.
	hasher, err := service.NewArgon2HasherWithParams(&argon2id.Params{
		Memory:      8 * 1024,
		Iterations:  1,
		Parallelism: 1,
		SaltLength:  16,
		KeyLength:   32,
	})
	if err != nil {
		t.Fatalf("hasher: %v", err)
	}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	svc := service.New(users, sessions, issuer, hasher, log, service.Config{
		AccessTTL:      15 * time.Minute,
		RefreshTTL:     12 * time.Hour,
		RefreshTTLLong: testTTL, // упрощаем проверку remember_me
	})
	return svc, users, sessions
}

func registerOK(t *testing.T, svc *service.Service) *service.Session {
	t.Helper()
	sess, err := svc.Register(context.Background(), "user@example.com", "sup3r-secret", ptr("Иван"))
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	return sess
}

func ptr[T any](v T) *T { return &v }
