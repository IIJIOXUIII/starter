package postgres_test

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rosterra/services/auth/internal/domain"
	"github.com/rosterra/services/auth/internal/repo/postgres"
)

// newTestPool поднимает postgres:18-alpine в testcontainers и гоняет миграции.
func newTestPool(t *testing.T) (context.CancelFunc, *postgres.UserRepo, *postgres.OutboxRepo) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "postgres:18-alpine",
			ExposedPorts: []string{"5432/tcp"},
			Env: map[string]string{
				"POSTGRES_USER":     "postgres",
				"POSTGRES_PASSWORD": "postgres",
				"POSTGRES_DB":       "auth_test",
			},
			// Двойное появление "ready" — PostgreSQL перезапускается после initdb.
			WaitingFor: wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(90 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cancel()
		t.Fatalf("testcontainer postgres: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
		cancel()
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	mapped, err := container.MappedPort(ctx, "5432/tcp")
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	raw := mapped.String()
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/auth_test?sslmode=disable", host, raw)

	log := slog.New(slog.NewTextHandler(testWriter{t}, nil))
	pool, err := postgres.NewPool(ctx, dsn, log)
	if err != nil {
		t.Fatalf("pool+migrations: %v", err)
	}
	return cancel, postgres.NewUserRepo(pool), postgres.NewOutboxRepo(pool)
}

type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Log(string(p))
	return len(p), nil
}

func TestUserRepoLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	_, users, outbox := newTestPool(t)
	ctx := context.Background()

	user := domain.NewUser("Anna@example.com", ptr("Anna"))
	event, err := buildEvent(domain.EventUserCreated)
	if err != nil {
		t.Fatal(err)
	}
	if err := users.Create(ctx, user, "$argon2id$fake-hash", &event); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Email citext — регистронезависимый поиск.
	got, err := users.GetByEmail(ctx, "anna@example.com")
	if err != nil {
		t.Fatalf("get by email: %v", err)
	}
	if got.ID != user.ID || got.Roles[0] != domain.RoleUser {
		t.Fatalf("неожиданный пользователь: %+v", got)
	}

	// Дубль email → уникальный индекс.
	err = users.Create(ctx, domain.NewUser("anna@example.com", nil), "x", nil)
	if err == nil {
		t.Fatal("ожидалась ошибка уникальности")
	}

	// GetByID.
	got, err = users.GetByID(ctx, user.ID)
	if err != nil || got.Name == nil || *got.Name != "Anna" {
		t.Fatalf("get by id: %v", err)
	}

	// Отсутствующий → ErrNotFound.
	if _, err := users.GetByID(ctx, uuid.New()); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидался ErrNotFound, получено %v", err)
	}

	// Статус + событие в той же транзакции.
	blockedEvent, _ := buildEvent(domain.EventUserBlocked)
	updated, err := users.UpdateStatus(ctx, user.ID, domain.StatusBlocked, &blockedEvent)
	if err != nil || updated.Status != domain.StatusBlocked {
		t.Fatalf("update status: %v", err)
	}

	// Роли.
	updated, err = users.SetRoles(ctx, user.ID, []domain.Role{domain.RoleUser, domain.RoleAdmin}, nil)
	if err != nil || len(updated.Roles) != 2 {
		t.Fatalf("set roles: %v", err)
	}

	// Список с фильтрами.
	items, total, err := users.List(ctx, domain.ListFilter{
		Query:   ptr("anna"),
		Status:  statusPtr(domain.StatusBlocked),
		Page:    1,
		PerPage: 20,
	})
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("list: items=%d total=%d err=%v", len(items), total, err)
	}
	admins, total, err := users.List(ctx, domain.ListFilter{Role: rolePtr(domain.RoleAdmin), Page: 1, PerPage: 20})
	if err != nil || total != 1 || len(admins) != 1 {
		t.Fatalf("list by role: %v", err)
	}

	// Outbox: user.created + user.blocked (изменения транзакционно с событиями).
	events, err := outbox.FetchUnpublished(ctx, 10)
	if err != nil || len(events) != 2 {
		t.Fatalf("outbox fetch: %d events, err=%v", len(events), err)
	}
	if events[0].Type != domain.EventUserCreated || events[1].Type != domain.EventUserBlocked {
		t.Fatalf("неожиданные события: %s, %s", events[0].Type, events[1].Type)
	}
	ids := make([]uuid.UUID, 0, len(events))
	for _, e := range events {
		ids = append(ids, e.ID)
	}
	if err := outbox.MarkPublished(ctx, ids); err != nil {
		t.Fatalf("mark published: %v", err)
	}
	events, _ = outbox.FetchUnpublished(ctx, 10)
	if len(events) != 0 {
		t.Fatalf("outbox должен быть пуст, осталось %d", len(events))
	}
}

func buildEvent(eventType string, data ...any) (domain.OutboxEvent, error) {
	return domain.OutboxEvent{ID: uuid.New(), Type: eventType, Payload: []byte(`{}`), CreatedAt: time.Now()}, nil
}

func statusPtr(s domain.UserStatus) *domain.UserStatus { return &s }
func rolePtr(r domain.Role) *domain.Role               { return &r }
func ptr[T any](v T) *T                                { return &v }
