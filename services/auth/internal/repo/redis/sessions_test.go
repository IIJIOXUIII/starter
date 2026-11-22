package redisrepo_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rosterra/services/auth/internal/domain"
	redisrepo "github.com/rosterra/services/auth/internal/repo/redis"
)

func newTestRepo(t *testing.T) (context.CancelFunc, *redisrepo.SessionRepo) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        "redis:8-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor:   wait.ForListeningPort("6379/tcp"),
		},
		Started: true,
	})
	if err != nil {
		cancel()
		t.Fatalf("testcontainer redis: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(context.Background())
		cancel()
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("host: %v", err)
	}
	mapped, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		t.Fatalf("port: %v", err)
	}
	// Port рендерится через Stringer как число; на всякий случай отрезаем протокол.
	raw := mapped.String()
	if i := strings.Index(raw, "/"); i >= 0 {
		raw = raw[:i]
	}
	rdb := redis.NewClient(&redis.Options{Addr: fmt.Sprintf("%s:%s", host, raw)})
	t.Cleanup(func() { _ = rdb.Close() })

	return cancel, redisrepo.NewSessionRepo(rdb)
}

func rec(userID uuid.UUID, family string, ttl time.Duration) domain.RefreshSession {
	now := time.Now().UTC()
	return domain.RefreshSession{
		UserID:    userID,
		SessionID: uuid.New(),
		Family:    family,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}

func TestSessionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	_, repo := newTestRepo(t)
	ctx := context.Background()

	uid := uuid.New()
	family := uuid.NewString()

	// Create + Get.
	t1, h1 := newToken()
	if err := repo.Create(ctx, rec(uid, family, time.Hour), h1); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := repo.Get(ctx, h1)
	if err != nil || got.UserID != uid || got.Family != family || got.ReplacedBy != "" {
		t.Fatalf("get: rec=%+v err=%v", got, err)
	}

	// Rotate: старый помечен заменённым, новый доступен.
	t2, h2 := newToken()
	if err := repo.Rotate(ctx, h1, rec(uid, family, time.Hour), h2); err != nil {
		t.Fatalf("rotate: %v", err)
	}
	old, err := repo.Get(ctx, h1)
	if err != nil || old.ReplacedBy != h2 {
		t.Fatalf("старая сессия должна быть помечена заменённой: %+v err=%v", old, err)
	}
	if _, err := repo.Get(ctx, h2); err != nil {
		t.Fatalf("новая сессия недоступна: %v", err)
	}

	// Reuse detection: использование заменённого токена отзывает семью целиком.
	_, h3 := newToken()
	if err := repo.Rotate(ctx, h1, rec(uid, family, time.Hour), h3); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("ожидался ErrSessionRevoked, получено %v", err)
	}
	if _, err := repo.Get(ctx, h2); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("семья должна быть отозвана, получено %v", err)
	}
	_ = t1
	_ = t2
}

func TestRevokeSingle(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	_, repo := newTestRepo(t)
	ctx := context.Background()

	uid := uuid.New()
	family := uuid.NewString()
	_, h := newToken()
	if err := repo.Create(ctx, rec(uid, family, time.Hour), h); err != nil {
		t.Fatal(err)
	}

	// Revoke чужого/несуществующего — идемпотентно.
	if err := repo.Revoke(ctx, "deadbeef"); err != nil {
		t.Fatalf("revoke unknown: %v", err)
	}

	if err := repo.Revoke(ctx, h); err != nil {
		t.Fatalf("revoke: %v", err)
	}
	if _, err := repo.Get(ctx, h); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("сессия должна быть удалена, получено %v", err)
	}
}

func TestRevokeAllForUser(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}
	_, repo := newTestRepo(t)
	ctx := context.Background()

	uid := uuid.New()
	// Две независимые сессии входа (две семьи).
	_, fa := newToken()
	if err := repo.Create(ctx, rec(uid, uuid.NewString(), time.Hour), fa); err != nil {
		t.Fatal(err)
	}
	_, fb := newToken()
	if err := repo.Create(ctx, rec(uid, uuid.NewString(), time.Hour), fb); err != nil {
		t.Fatal(err)
	}

	// Чужая сессия не должна пострадать.
	other := uuid.New()
	_, hOther := newToken()
	if err := repo.Create(ctx, rec(other, uuid.NewString(), time.Hour), hOther); err != nil {
		t.Fatal(err)
	}

	if err := repo.RevokeAllForUser(ctx, uid); err != nil {
		t.Fatalf("revoke all: %v", err)
	}
	for _, h := range []string{fa, fb} {
		if _, err := repo.Get(ctx, h); !errors.Is(err, domain.ErrInvalidRefreshToken) {
			t.Fatalf("сессия %s должна быть отозвана", h)
		}
	}
	if _, err := repo.Get(ctx, hOther); err != nil {
		t.Fatalf("чужая сессия не должна затрагиваться: %v", err)
	}
}

// newToken — тестовый opaque-токен + hex-хеш (как в сервисе).
func newToken() (token, hash string) {
	token = uuid.NewString() + "-token"
	return token, domain.TokenHashHex(token)
}
