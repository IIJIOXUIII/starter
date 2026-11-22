// Package jwtx — выпуск и проверка JWT (EdDSA, Ed25519) + JWKS.
//
// Один Issuer живёт в auth-service (владеет приватным ключом).
// Другие сервисы используют Verifier (только публичный ключ + JWKS-кеш).
//
// Модель ключей рассчитана на ротацию: kid выводится из публичного ключа,
// поэтому смена ключа автоматически обновляет набор JWKS.
package jwtx

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Ошибки проверки токена.
var (
	ErrTokenInvalid = errors.New("jwtx: token invalid")
	ErrTokenExpired = errors.New("jwtx: token expired")
)

// Issuer выпускает access-токены. Владелец приватного ключа.
type Issuer struct {
	key ed25519.PrivateKey
	kid string
	iss string
	aud string
	ttl time.Duration
	now func() time.Time
}

// Claims — полезная нагрузка access-токена.
type Claims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	Email     string
	Roles     []string
}

// NewIssuer создаёт Issuer из seed (64 байта ключа в base64/std или hex).
// Пустой seed → генерируется эфемерный ключ (только для dev; сервис логирует предупреждение).
func NewIssuer(seed, iss, aud string, ttl time.Duration) (*Issuer, bool /*ephemeral*/, error) {
	var key ed25519.PrivateKey
	ephemeral := seed == ""

	if ephemeral {
		_, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			return nil, false, fmt.Errorf("jwtx: generate key: %w", err)
		}
		key = priv
	} else {
		raw, err := decodeSeed(seed)
		if err != nil {
			return nil, false, fmt.Errorf("jwtx: seed: %w", err)
		}
		if len(raw) != ed25519.SeedSize {
			return nil, false, fmt.Errorf("jwtx: seed must be %d bytes, got %d", ed25519.SeedSize, len(raw))
		}
		key = ed25519.NewKeyFromSeed(raw)
	}

	return &Issuer{
		key: key,
		kid: KeyID(key.Public().(ed25519.PublicKey)),
		iss: iss,
		aud: aud,
		ttl: ttl,
		now: time.Now,
	}, ephemeral, nil
}

func decodeSeed(seed string) ([]byte, error) {
	if raw, err := hex.DecodeString(seed); err == nil {
		return raw, nil
	}
	return base64.StdEncoding.DecodeString(seed)
}

// KeyID — стабильный идентификатор ключа (kid) из его публичной части.
func KeyID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:8])
}

// Issue подписывает access-токен для пользователя в рамках сессии.
func (i *Issuer) Issue(userID, sessionID uuid.UUID, email string, roles []string) (token string, expiresIn time.Duration, err error) {
	now := i.now()
	claims := jwt.MapClaims{
		"iss":   i.iss,
		"aud":   i.aud,
		"sub":   userID.String(),
		"sid":   sessionID.String(),
		"email": email,
		"roles": roles,
		"iat":   now.Unix(),
		"exp":   now.Add(i.ttl).Unix(),
		"jti":   uuid.NewString(),
	}
	token, err = jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims).SignedString(i.key)
	if err != nil {
		return "", 0, fmt.Errorf("jwtx: sign: %w", err)
	}
	return token, i.ttl, nil
}

// JWKS возвращает JSON-набор публичных ключей (RFC 7517).
func (i *Issuer) JWKS() json.RawMessage {
	set := map[string]any{"keys": i.JWKSKeys()}
	raw, _ := json.Marshal(set)
	return raw
}

// JWKSKeys возвращает публичные ключи в виде структур — для strict-контракта
// (gen.JsonWebKeySet.Keys).
func (i *Issuer) JWKSKeys() []map[string]any {
	pub := i.key.Public().(ed25519.PublicKey)
	return []map[string]any{{
		"kty": "OKP",
		"crv": "Ed25519",
		"use": "sig",
		"alg": "EdDSA",
		"kid": i.kid,
		"x":   base64.RawURLEncoding.EncodeToString(pub),
	}}
}

// Kid — идентификатор текущего ключа.
func (i *Issuer) Kid() string { return i.kid }

// Verifier проверяет подпись и структуру токена без приватного ключа.
type Verifier struct {
	pub ed25519.PublicKey
	kid string
	iss string
	aud string
	now func() time.Time
}

// NewVerifier создаёт проверяльщик из seed (тот же, что у Issuer) —
// в будущем: из JWKS auth-service или конфига публичных ключей.
func NewVerifier(seed, iss, aud string) (*Verifier, error) {
	raw, err := decodeSeed(seed)
	if err != nil {
		return nil, fmt.Errorf("jwtx: seed: %w", err)
	}
	if len(raw) != ed25519.SeedSize {
		return nil, fmt.Errorf("jwtx: seed must be %d bytes, got %d", ed25519.SeedSize, len(raw))
	}
	key := ed25519.NewKeyFromSeed(raw)
	return &Verifier{
		pub: key.Public().(ed25519.PublicKey),
		kid: KeyID(key.Public().(ed25519.PublicKey)),
		iss: iss,
		aud: aud,
		now: time.Now,
	}, nil
}

// Verify разбирает токен, проверяет подпись, iss/aud и срок.
func (v *Verifier) Verify(token string) (*Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("%w: unexpected alg %q", ErrTokenInvalid, t.Header["alg"])
		}
		return v.pub, nil
	}, jwt.WithIssuer(v.iss), jwt.WithAudience(v.aud), jwt.WithExpirationRequired())
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !parsed.Valid {
		return nil, ErrTokenInvalid
	}
	mc, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenInvalid
	}
	uid, err := uuid.Parse(str(mc["sub"]))
	if err != nil {
		return nil, ErrTokenInvalid
	}
	sid, err := uuid.Parse(str(mc["sid"]))
	if err != nil {
		return nil, ErrTokenInvalid
	}
	roles, _ := mc["roles"].([]any)
	out := make([]string, 0, len(roles))
	for _, r := range roles {
		if s, ok := r.(string); ok {
			out = append(out, s)
		}
	}
	return &Claims{UserID: uid, SessionID: sid, Email: str(mc["email"]), Roles: out}, nil
}

// Kid — идентификатор ключа проверки (для отладки и ротации).
func (v *Verifier) Kid() string { return v.kid }

func str(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
