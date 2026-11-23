// Package service — бизнес-логика auth-service: сессии, токены, RBAC.
//
// Сервис зависит только от портов домена (domain.UserRepo/SessionRepo/...)
// и интерфейса выпускающего токены — тестируется на подменах.
package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/rosterra/services/auth/internal/domain"
)

// TokenIssuer — выпуск access-JWT (реализуется адаптером над pkg/jwtx).
type TokenIssuer interface {
	Issue(userID, sessionID uuid.UUID, email string, roles []string) (token string, ttl time.Duration, err error)
}

// Tokens — пара токенов с TTL (возвращается транспортом как SessionTokens контракта).
type Tokens struct {
	AccessToken  string
	AccessTTL    time.Duration
	RefreshToken string
	RefreshTTL   time.Duration
}

// Session — результат выдачи сессии.
type Session struct {
	Tokens      Tokens
	User        *domain.User
	RefreshHash string // hex-хеш refresh-токена (ключ в Redis)
}

// Service — фасад бизнес-логики.
type Service struct {
	users       domain.UserRepo
	sessions    domain.SessionRepo
	issuer      TokenIssuer
	hasher      PasswordHasher
	log         *slog.Logger
	accessTTL   time.Duration
	refreshTTL  time.Duration // при remember_me=false
	refreshLong time.Duration // при remember_me=true
}

// Config — параметры сервиса.
type Config struct {
	AccessTTL      time.Duration // TTL access-токена (15m)
	RefreshTTL     time.Duration // TTL refresh при remember_me=false (12h)
	RefreshTTLLong time.Duration // TTL refresh при remember_me=true (30d)
}

// New конструирует сервис.
func New(users domain.UserRepo, sessions domain.SessionRepo, issuer TokenIssuer, hasher PasswordHasher, log *slog.Logger, cfg Config) *Service {
	return &Service{
		users:       users,
		sessions:    sessions,
		issuer:      issuer,
		hasher:      hasher,
		log:         log,
		accessTTL:   cfg.AccessTTL,
		refreshTTL:  cfg.RefreshTTL,
		refreshLong: cfg.RefreshTTLLong,
	}
}

// Ошибки валидации входа — транспорт маппит их в 400 validation_failed с details.
type fieldError struct {
	Field   string
	Message string
}

// ValidateError — ошибка валидации с полями для error-envelope details.
type ValidateError struct {
	Fields []fieldError
}

func (e *ValidateError) Error() string {
	if len(e.Fields) == 0 {
		return "validation failed"
	}
	return fmt.Sprintf("validation failed: %s %s", e.Fields[0].Field, e.Fields[0].Message)
}

func validationError(field, msg string) *ValidateError {
	return &ValidateError{Fields: []fieldError{{Field: field, Message: msg}}}
}

// Register регистрирует пользователя: Argon2id + транзакция (user+role+outbox)
// + новая refresh-сессия + access-токен (авто-логин).
func (s *Service) Register(ctx context.Context, email, password string, name *string) (*Session, error) {
	if verr := validateCredentials(email, password, name); verr != nil {
		return nil, verr
	}

	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, domain.ErrEmailTaken
	} else if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("lookup user: %w", err)
	}

	hash, err := s.hasher.Hash(password)
	if err != nil {
		return nil, err
	}

	user := domain.NewUser(email, name)
	event, err := buildEvent(ctx, domain.EventUserCreated, toPublicUser(user))
	if err != nil {
		return nil, err
	}
	if err := s.users.Create(ctx, user, hash, &event); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	s.log.Info("user registered", "user_id", user.ID)
	return s.startSession(ctx, user, s.refreshLong)
}

// Login аутентифицирует по email и паролю.
// Несуществующий email обрабатывается так же, как неверный пароль (timing-safety).
func (s *Service) Login(ctx context.Context, email, password string, rememberMe bool) (*Session, error) {
	if email == "" || password == "" {
		return nil, validationError("email", "email and password are required")
	}

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		s.hasher.VerifyDummy(password)
		return nil, domain.ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("lookup user: %w", err)
	}
	if user.Status == domain.StatusBlocked {
		return nil, domain.ErrAccountLocked
	}
	if !s.hasher.Verify(user.PasswordHash, password) {
		return nil, domain.ErrInvalidCredentials
	}

	ttl := s.refreshTTL
	if rememberMe {
		ttl = s.refreshLong
	}
	s.log.Info("user logged in", "user_id", user.ID)
	return s.startSession(ctx, user, ttl)
}

// Refresh ротирует refresh-токен и выпускает новый access.
// Повторное использование ротированного токена отзывает всю семью (reuse detection).
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Session, error) {
	if refreshToken == "" {
		return nil, domain.ErrInvalidRefreshToken
	}
	oldHash := domain.TokenHashHex(refreshToken)

	rec, err := s.sessions.Get(ctx, oldHash)
	if err != nil {
		return nil, err
	}

	user, err := s.users.GetByID(ctx, rec.UserID)
	if errors.Is(err, domain.ErrNotFound) {
		_ = s.sessions.RevokeFamily(ctx, rec.Family)
		return nil, domain.ErrInvalidRefreshToken
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	if user.Status == domain.StatusBlocked {
		_ = s.sessions.RevokeFamily(ctx, rec.Family)
		return nil, domain.ErrAccountLocked
	}

	newToken, newHash, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	newRec := newSessionRecord(user.ID, rec.Family, time.Until(rec.ExpiresAt))
	if err := s.sessions.Rotate(ctx, oldHash, newRec, newHash); err != nil {
		return nil, err
	}

	access, ttl, err := s.issuer.Issue(user.ID, newRec.SessionID, user.Email, rolesToStrings(user.Roles))
	if err != nil {
		return nil, err
	}
	return &Session{
		Tokens: Tokens{
			AccessToken:  access,
			AccessTTL:    ttl,
			RefreshToken: newToken,
			RefreshTTL:   time.Until(newRec.ExpiresAt),
		},
		User:        user,
		RefreshHash: newHash,
	}, nil
}

// Logout отзывает refresh-сессию (идемпотентно).
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.sessions.Revoke(ctx, domain.TokenHashHex(refreshToken))
}

// Me — текущий пользователь по данным access-токена.
func (s *Service) Me(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user.Status == domain.StatusBlocked {
		return nil, domain.ErrAccountLocked
	}
	return user, nil
}

// startSession создаёт refresh-сессию и выпускает токены.
func (s *Service) startSession(ctx context.Context, user *domain.User, refreshTTL time.Duration) (*Session, error) {
	refreshToken, hash, err := newRefreshToken()
	if err != nil {
		return nil, err
	}
	rec := newSessionRecord(user.ID, uuid.NewString(), refreshTTL)
	if err := s.sessions.Create(ctx, rec, hash); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	access, ttl, err := s.issuer.Issue(user.ID, rec.SessionID, user.Email, rolesToStrings(user.Roles))
	if err != nil {
		return nil, err
	}
	return &Session{
		Tokens: Tokens{
			AccessToken:  access,
			AccessTTL:    ttl,
			RefreshToken: refreshToken,
			RefreshTTL:   refreshTTL,
		},
		User:        user,
		RefreshHash: hash,
	}, nil
}

func newSessionRecord(userID uuid.UUID, family string, ttl time.Duration) domain.RefreshSession {
	now := time.Now().UTC()
	return domain.RefreshSession{
		UserID:    userID,
		SessionID: uuid.New(),
		Family:    family,
		CreatedAt: now,
		ExpiresAt: now.Add(ttl),
	}
}

// newRefreshToken — 32 случайных байта, base64url (opaque; в Redis только SHA-256).
func newRefreshToken() (token, hashHex string, err error) {
	buf := make([]byte, 32)
	if _, err = rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("random refresh token: %w", err)
	}
	token = base64.RawURLEncoding.EncodeToString(buf)
	return token, domain.TokenHashHex(token), nil
}

func rolesToStrings(roles []domain.Role) []string {
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		out = append(out, string(r))
	}
	return out
}

// validateCredentials — инварианты контракта (email/password/name).
func validateCredentials(email, password string, name *string) error {
	var verr *ValidateError
	if email == "" || !isEmail(email) {
		verr = addField(verr, "email", "must be a valid email address")
	}
	if len(password) < 8 {
		verr = addField(verr, "password", "must be at least 8 characters long")
	}
	if len(password) > 128 {
		verr = addField(verr, "password", "must be at most 128 characters long")
	}
	if name != nil && len(*name) > 100 {
		verr = addField(verr, "name", "must be at most 100 characters long")
	}
	// Важно: без этой проверки typed-nil *ValidateError станет ненулевым error-интерфейсом.
	if verr == nil {
		return nil
	}
	return verr
}

// isEmail — pragmatic-проверка формата (полная валидация — в BFF через zod).
func isEmail(s string) bool {
	at := -1
	for i, c := range s {
		if c == '@' {
			if at != -1 {
				return false
			}
			at = i
		} else if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
			c == '.' || c == '-' || c == '_' || c == '+') {
			return false
		}
	}
	return at > 0 && at < len(s)-1
}

func addField(err *ValidateError, field, msg string) *ValidateError {
	if err == nil {
		err = &ValidateError{}
	}
	err.Fields = append(err.Fields, fieldError{Field: field, Message: msg})
	return err
}
