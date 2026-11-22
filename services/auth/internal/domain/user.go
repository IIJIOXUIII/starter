// Package domain — сущности auth-service и порты (интерфейсы инфраструктуры).
//
// Домен не знает про pgx, redis, nats и HTTP: зависимости направлены только
// внутрь (transport/repo → domain). Ошибки домена — sentinel'ы, транспорт
// маппит их в HTTP-коды и error-envelope контракта.
package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Роли и статусы.
const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"

	StatusActive  UserStatus = "active"
	StatusBlocked UserStatus = "blocked"
)

// Роль пользователя в RBAC.
type Role string

// Статус пользователя.
type UserStatus string

// Доменные ошибки — маппятся на коды error-envelope контракта.
var (
	ErrEmailTaken          = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid credentials")
	ErrAccountLocked       = errors.New("account is blocked")
	ErrNotFound            = errors.New("not found")
	ErrSessionRevoked      = errors.New("refresh session revoked (reuse detected)")
	ErrInvalidRefreshToken = errors.New("refresh token invalid")
	ErrForbidden           = errors.New("insufficient permissions")
	ErrValidation          = errors.New("validation failed")
)

// User — пользователь auth-service (credentials, статус, роли).
type User struct {
	ID            uuid.UUID
	Email         string
	Name          *string
	Status        UserStatus
	Roles         []Role
	EmailVerified bool
	PasswordHash  string // никогда не покидает сервис
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// HasRole проверяет наличие роли.
func (u *User) HasRole(r Role) bool {
	for _, have := range u.Roles {
		if have == r {
			return true
		}
	}
	return false
}

// NewUser создаёт пользователя по умолчанию: active, роль user, email не подтверждён.
func NewUser(email string, name *string) *User {
	now := time.Now().UTC()
	return &User{
		ID:            uuid.New(),
		Email:         email,
		Name:          name,
		Status:        StatusActive,
		Roles:         []Role{RoleUser},
		EmailVerified: false,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

// RefreshSession — refresh-сессия (хранится в Redis, тут — чистая структура).
type RefreshSession struct {
	UserID     uuid.UUID
	SessionID  uuid.UUID // попадает в claim sid access-токена
	Family     string    // цепочка ротации: все токены одной сессии входа
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ReplacedBy string // hex-хеш токена, на который выполнена ротация
	UserAgent  string
	IP         string
}

// TokenHashHex — hex SHA-256 refresh-токена; ключ сессии в Redis.
func TokenHashHex(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
