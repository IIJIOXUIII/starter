// Package redisrepo — реализация domain.SessionRepo на Redis.
//
// Структура ключей:
//
//	sess:{token_hash_hex}  → JSON RefreshSession, TTL = до истечения сессии
//	fam:{family_uuid}      → set hash'ей сессий семьи ротации
//	usr:{user_id}          → set семей пользователя (для revoke-all при блокировке)
//
// Ротация под WATCH: конкурентное использование одного refresh-токена
// детерминированно приводит к ErrSessionRevoked у проигравшего гонку.
package redisrepo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/rosterra/services/auth/internal/domain"
)

// SessionRepo — Redis-реализация domain.SessionRepo.
type SessionRepo struct {
	rdb *redis.Client
}

// NewSessionRepo конструирует репозиторий сессий.
func NewSessionRepo(rdb *redis.Client) *SessionRepo {
	return &SessionRepo{rdb: rdb}
}

func sessKey(hash string) string  { return "sess:" + hash }
func famKey(fam string) string    { return "fam:" + fam }
func usrKey(uid uuid.UUID) string { return "usr:" + uid.String() }

func ttlUntil(t time.Time) time.Duration {
	d := time.Until(t)
	if d <= 0 {
		return time.Minute // защита от нулевого TTL при истёкшем времени
	}
	return d
}

// Create регистрирует новую сессию и индексы (семья, пользователь).
func (r *SessionRepo) Create(ctx context.Context, rec domain.RefreshSession, tokenHash string) error {
	raw, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	ttl := ttlUntil(rec.ExpiresAt)
	pipe := r.rdb.Pipeline()
	pipe.Set(ctx, sessKey(tokenHash), raw, ttl)
	pipe.SAdd(ctx, famKey(rec.Family), tokenHash)
	pipe.Expire(ctx, famKey(rec.Family), ttl)
	pipe.SAdd(ctx, usrKey(rec.UserID), rec.Family)
	pipe.Expire(ctx, usrKey(rec.UserID), ttl)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

// Get возвращает сессию по хешу refresh-токена.
func (r *SessionRepo) Get(ctx context.Context, tokenHash string) (domain.RefreshSession, error) {
	raw, err := r.rdb.Get(ctx, sessKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return domain.RefreshSession{}, domain.ErrInvalidRefreshToken
	}
	if err != nil {
		return domain.RefreshSession{}, fmt.Errorf("get session: %w", err)
	}
	var rec domain.RefreshSession
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return domain.RefreshSession{}, fmt.Errorf("unmarshal session: %w", err)
	}
	return rec, nil
}

// Rotate атомарно помечает старый токен заменённым и создаёт новый в той же семье.
// Если токен уже был заменён (reuse) — возвращает ErrSessionRevoked и отзывает семью.
func (r *SessionRepo) Rotate(ctx context.Context, oldHash string, newRec domain.RefreshSession, newHash string) error {
	newRaw, err := json.Marshal(newRec)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	err = r.rdb.Watch(ctx, func(tx *redis.Tx) error {
		raw, err := tx.Get(ctx, sessKey(oldHash)).Result()
		if errors.Is(err, redis.Nil) {
			return domain.ErrInvalidRefreshToken
		}
		if err != nil {
			return fmt.Errorf("get session: %w", err)
		}
		var rec domain.RefreshSession
		if err := json.Unmarshal([]byte(raw), &rec); err != nil {
			return fmt.Errorf("unmarshal session: %w", err)
		}
		if rec.ReplacedBy != "" {
			// Refresh-токен использован повторно — признак кражи: отзываем всю цепочку.
			return domain.ErrSessionRevoked
		}

		rec.ReplacedBy = newHash
		mutated, err := json.Marshal(rec)
		if err != nil {
			return fmt.Errorf("marshal session: %w", err)
		}

		newTTL := ttlUntil(newRec.ExpiresAt)
		_, err = tx.TxPipelined(ctx, func(pipe redis.Pipeliner) error {
			pipe.Set(ctx, sessKey(oldHash), mutated, ttlUntil(rec.ExpiresAt))
			pipe.Set(ctx, sessKey(newHash), newRaw, newTTL)
			pipe.SAdd(ctx, famKey(newRec.Family), newHash)
			pipe.Expire(ctx, famKey(newRec.Family), newTTL)
			return nil
		})
		return err
	}, sessKey(oldHash))

	if err != nil {
		if errors.Is(err, domain.ErrSessionRevoked) {
			_ = r.RevokeFamily(ctx, newRec.Family)
			return domain.ErrSessionRevoked
		}
		if errors.Is(err, domain.ErrInvalidRefreshToken) {
			return domain.ErrInvalidRefreshToken
		}
		return fmt.Errorf("rotate: %w", err)
	}
	return nil
}

// Revoke удаляет сессию (logout) и чистит индексы опустевшей семьи.
func (r *SessionRepo) Revoke(ctx context.Context, tokenHash string) error {
	raw, err := r.rdb.Get(ctx, sessKey(tokenHash)).Result()
	if errors.Is(err, redis.Nil) {
		return nil // уже отозвана — идемпотентно
	}
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	var rec domain.RefreshSession
	if err := json.Unmarshal([]byte(raw), &rec); err != nil {
		return fmt.Errorf("unmarshal session: %w", err)
	}

	pipe := r.rdb.Pipeline()
	pipe.Del(ctx, sessKey(tokenHash))
	pipe.SRem(ctx, famKey(rec.Family), tokenHash)
	card, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	_ = card

	// Если семья опустела — удаляем её и индекс пользователя.
	if n, err := r.rdb.SCard(ctx, famKey(rec.Family)).Result(); err == nil && n == 0 {
		pipe = r.rdb.Pipeline()
		pipe.Del(ctx, famKey(rec.Family))
		pipe.SRem(ctx, usrKey(rec.UserID), rec.Family)
		if _, err := pipe.Exec(ctx); err != nil {
			return fmt.Errorf("cleanup family: %w", err)
		}
	}
	return nil
}

// RevokeFamily отзывает всю цепочку ротации (reuse detection).
func (r *SessionRepo) RevokeFamily(ctx context.Context, family string) error {
	hashes, err := r.rdb.SMembers(ctx, famKey(family)).Result()
	if err != nil {
		return fmt.Errorf("members family: %w", err)
	}

	// Определяем владельца до удаления сессий.
	var userID uuid.UUID
	for _, h := range hashes {
		rec, err := r.Get(ctx, h)
		if err == nil {
			userID = rec.UserID
			break
		}
	}

	keys := make([]string, 0, len(hashes)+1)
	for _, h := range hashes {
		keys = append(keys, sessKey(h))
	}
	keys = append(keys, famKey(family))
	if err := r.rdb.Del(ctx, keys...).Err(); err != nil {
		return fmt.Errorf("del family: %w", err)
	}
	if userID != uuid.Nil {
		if err := r.rdb.SRem(ctx, usrKey(userID), family).Err(); err != nil {
			return fmt.Errorf("cleanup user index: %w", err)
		}
	}
	return nil
}

// RevokeAllForUser отзывает все сессии пользователя (блокировка аккаунта).
func (r *SessionRepo) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	families, err := r.rdb.SMembers(ctx, usrKey(userID)).Result()
	if err != nil {
		return fmt.Errorf("members user: %w", err)
	}
	for _, fam := range families {
		if err := r.RevokeFamily(ctx, fam); err != nil {
			return err
		}
	}
	if err := r.rdb.Del(ctx, usrKey(userID)).Err(); err != nil {
		return fmt.Errorf("del user index: %w", err)
	}
	return nil
}
