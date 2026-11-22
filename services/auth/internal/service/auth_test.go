package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/services/auth/internal/domain"
	"github.com/rosterra/services/auth/internal/service"
)

func TestRegisterSuccess(t *testing.T) {
	svc, users, sessions := newTestService(t)

	sess := registerOK(t, svc)

	if sess.Tokens.AccessToken == "" || sess.Tokens.RefreshToken == "" {
		t.Fatal("ожидались access и refresh токены")
	}
	if sess.User.ID == uuid.Nil {
		t.Fatal("ожидался id пользователя")
	}

	// Пользователь сохранён с ролью user и не подтверждённым email.
	u, err := users.GetByEmail(context.Background(), "user@example.com")
	if err != nil {
		t.Fatalf("user not found: %v", err)
	}
	if len(u.Roles) != 1 || u.Roles[0] != domain.RoleUser {
		t.Fatalf("ожидалась роль user, получено %v", u.Roles)
	}
	if u.Status != domain.StatusActive || u.EmailVerified {
		t.Fatalf("неожиданный статус/флаги: %s verified=%v", u.Status, u.EmailVerified)
	}
	if u.PasswordHash == "" || u.PasswordHash == "sup3r-secret" {
		t.Fatal("пароль должен быть Argon2id-хешем")
	}

	// Событие user.created ушло в outbox.
	if len(users.events) != 1 || users.events[0] != domain.EventUserCreated {
		t.Fatalf("ожидалось user.created, получено %v", users.events)
	}

	// Сессия доступна в хранилище по хешу refresh-токена.
	if _, err := sessions.Get(context.Background(), domain.TokenHashHex(sess.Tokens.RefreshToken)); err != nil {
		t.Fatalf("сессия не найдена: %v", err)
	}
}

func TestRegisterDuplicateEmail(t *testing.T) {
	svc, _, _ := newTestService(t)
	registerOK(t, svc)

	_, err := svc.Register(context.Background(), "USER@example.com", "another-pass1", nil)
	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("ожидался ErrEmailTaken, получено %v", err)
	}
}

func TestRegisterValidation(t *testing.T) {
	svc, _, _ := newTestService(t)

	_, err := svc.Register(context.Background(), "bad-email", "sup3r-secret", nil)
	var verr *service.ValidateError
	if err == nil || !errors.As(err, &verr) {
		t.Fatalf("ожидалась ValidateError для email, получено %v", err)
	}

	_, err = svc.Register(context.Background(), "ok@example.com", "short", nil)
	var ve *service.ValidateError
	if err == nil || !errors.As(err, &ve) {
		t.Fatalf("ожидалась ValidateError для пароля, получено %v", err)
	}
	if len(ve.Fields) != 1 || ve.Fields[0].Field != "password" {
		t.Fatalf("ожидалось поле password, получено %+v", ve.Fields)
	}
}

func TestLoginFlow(t *testing.T) {
	svc, users, _ := newTestService(t)
	registerOK(t, svc)

	// Успешный вход.
	sess, err := svc.Login(context.Background(), "user@example.com", "sup3r-secret", true)
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	// remember_me=true → долгий TTL (в тесте RefreshTTLLong = testTTL).
	if sess.Tokens.RefreshTTL != testTTL {
		t.Fatalf("ожидался длинный refresh TTL, получено %v", sess.Tokens.RefreshTTL)
	}

	// Короткий TTL без remember_me.
	sess2, err := svc.Login(context.Background(), "user@example.com", "sup3r-secret", false)
	if err != nil {
		t.Fatalf("login (short): %v", err)
	}
	if sess2.Tokens.RefreshTTL != 12*time.Hour {
		t.Fatalf("ожидался refresh TTL 12h, получено %v", sess2.Tokens.RefreshTTL)
	}

	// Неверный пароль.
	if _, err := svc.Login(context.Background(), "user@example.com", "wrong-password", false); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("ожидался ErrInvalidCredentials, получено %v", err)
	}

	// Несуществующий email — та же ошибка (без раскрытия существования).
	if _, err := svc.Login(context.Background(), "ghost@example.com", "whatever1", false); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("ожидался ErrInvalidCredentials, получено %v", err)
	}

	// Блокировка → вход запрещён.
	u, _ := users.GetByEmail(context.Background(), "user@example.com")
	_, _ = svc.SetStatus(context.Background(), u.ID, domain.StatusBlocked, ptr("нарушение"))
	if _, err := svc.Login(context.Background(), "user@example.com", "sup3r-secret", false); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("ожидался ErrAccountLocked, получено %v", err)
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	ctx := context.Background()
	svc, _, sessions := newTestService(t)
	sess := registerOK(t, svc)

	// Первая ротация: новый refresh != старый.
	s2, err := svc.Refresh(ctx, sess.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if s2.Tokens.RefreshToken == sess.Tokens.RefreshToken {
		t.Fatal("refresh-токен должен ротироваться")
	}

	// Повторное использование старого токена → reuse detection → отзыв семьи.
	if _, err := svc.Refresh(ctx, sess.Tokens.RefreshToken); !errors.Is(err, domain.ErrSessionRevoked) {
		t.Fatalf("ожидался ErrSessionRevoked, получено %v", err)
	}

	// Токен из той же семьи больше не действует.
	if _, err := svc.Refresh(ctx, s2.Tokens.RefreshToken); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("ожидался ErrInvalidRefreshToken, получено %v", err)
	}

	// В хранилище не осталось ключей семьи.
	if len(sessions.sessions) != 0 {
		t.Fatalf("семья должна быть полностью отозвана, осталось %d", len(sessions.sessions))
	}
}

func TestLogoutInvalidatesSession(t *testing.T) {
	ctx := context.Background()
	svc, _, sessions := newTestService(t)
	sess := registerOK(t, svc)

	if err := svc.Logout(ctx, sess.Tokens.RefreshToken); err != nil {
		t.Fatalf("logout: %v", err)
	}
	if _, err := svc.Refresh(ctx, sess.Tokens.RefreshToken); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("после logout refresh должен быть недействителен, получено %v", err)
	}
	if len(sessions.sessions) != 0 {
		t.Fatalf("ключ сессии должен быть удалён, осталось %d", len(sessions.sessions))
	}
}

func TestMeAndAdminRBAC(t *testing.T) {
	ctx := context.Background()
	svc, users, sessions := newTestService(t)
	sess := registerOK(t, svc)

	// Me по данным токена.
	me, err := svc.Me(ctx, sess.User.ID)
	if err != nil || me.Email != "user@example.com" {
		t.Fatalf("me: %v", err)
	}

	// Обновляем сессию ротацией ДО блокировки.
	s2, err := svc.Refresh(ctx, sess.Tokens.RefreshToken)
	if err != nil {
		t.Fatalf("refresh before block: %v", err)
	}

	// Me заблокированного — запрещён.
	_, _ = svc.SetStatus(ctx, sess.User.ID, domain.StatusBlocked, nil)
	if _, err := svc.Me(ctx, sess.User.ID); !errors.Is(err, domain.ErrAccountLocked) {
		t.Fatalf("ожидался ErrAccountLocked, получено %v", err)
	}

	// Сессии отозваны из хранилища → refresh отклоняется как невалидный.
	if _, err := svc.Refresh(ctx, s2.Tokens.RefreshToken); !errors.Is(err, domain.ErrInvalidRefreshToken) {
		t.Fatalf("после блокировки сессии должны отзываться, получено %v", err)
	}
	if len(sessions.sessions) != 0 {
		t.Fatalf("все сессии должны быть отозваны, осталось %d", len(sessions.sessions))
	}
	_, _ = svc.SetStatus(ctx, sess.User.ID, domain.StatusActive, nil)

	// Повторный вход после разблокировки.
	s3, err := svc.Login(ctx, "user@example.com", "sup3r-secret", false)
	if err != nil {
		t.Fatalf("login after unblock: %v", err)
	}

	// Выдача роли admin отражается в хранилище.
	u, err := svc.SetRoles(ctx, sess.User.ID, []domain.Role{domain.RoleAdmin})
	if err != nil {
		t.Fatalf("set roles: %v", err)
	}
	if !u.HasRole(domain.RoleAdmin) {
		t.Fatal("ожидалась роль admin")
	}
	_ = s3

	// События статуса и ролей зафиксированы в outbox.
	found := map[string]bool{}
	for _, e := range users.events {
		found[e] = true
	}
	for _, want := range []string{domain.EventUserBlocked, domain.EventUserUnblocked, domain.EventRoleChanged} {
		if !found[want] {
			t.Fatalf("не найдено событие %s в %v", want, users.events)
		}
	}
}

func TestAccessTokenClaims(t *testing.T) {
	const seed = "a2b4c6d8e0f2a4b6c8d0e2f4a6b8c0d2e4f6a8b0c2d4e6f8a0b2c4d6e8f0a2b4"
	verifier, err := jwtx.NewVerifier(seed, "rosterra-auth", "rosterra")
	if err != nil {
		t.Fatalf("verifier: %v", err)
	}
	svc, _, _ := newTestService(t)
	sess := registerOK(t, svc)

	claims, err := verifier.Verify(sess.Tokens.AccessToken)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != sess.User.ID || claims.Email != "user@example.com" {
		t.Fatalf("неожиданные claims: %+v", claims)
	}
	if len(claims.Roles) != 1 || claims.Roles[0] != "user" {
		t.Fatalf("ожидалась роль user в claims, получено %v", claims.Roles)
	}

	// Подделка подписи/чужой токен → отказ.
	if _, err := verifier.Verify(sess.Tokens.AccessToken + "x"); err == nil {
		t.Fatal("подделанный токен должен отвергаться")
	}
}
