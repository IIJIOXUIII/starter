package service

import (
	"fmt"

	"github.com/alexedwards/argon2id"
)

// PasswordHasher — хеширование и проверка паролей (Argon2id).
type PasswordHasher interface {
	// Hash возвращает PHC-строку argon2id (параметры внутри строки).
	Hash(password string) (string, error)
	// Verify — constant-time проверка пароля против хеша.
	Verify(hash, password string) bool
	// VerifyDummy — constant-time проверка против эталонного хеша: вызывается,
	// когда пользователя с таким email нет, чтобы время ответа login
	// не раскрывало существование email.
	VerifyDummy(password string)
}

// Argon2Hasher — реализация на github.com/alexedwards/argon2id с OWASP-параметрами.
type Argon2Hasher struct {
	params    *argon2id.Params
	dummyHash string // для login несуществующего пользователя: защита от timing-enumeration
}

// NewArgon2Hasher создаёт хешер. Параметры по умолчанию OWASP:
// memory 64 MB, iterations 3, parallelism 4.
func NewArgon2Hasher() (*Argon2Hasher, error) {
	return NewArgon2HasherWithParams(argon2id.DefaultParams)
}

// NewArgon2HasherWithParams — хешер с явными параметрами (для тестов — ослабленные).
func NewArgon2HasherWithParams(params *argon2id.Params) (*Argon2Hasher, error) {
	h := &Argon2Hasher{params: params}
	dummy, err := argon2id.CreateHash("rosterra-timing-dummy", params)
	if err != nil {
		return nil, fmt.Errorf("create dummy hash: %w", err)
	}
	h.dummyHash = dummy
	return h, nil
}

// Hash хеширует пароль.
func (h *Argon2Hasher) Hash(password string) (string, error) {
	hash, err := argon2id.CreateHash(password, h.params)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return hash, nil
}

// Verify проверяет пароль против хеша.
func (h *Argon2Hasher) Verify(hash, password string) bool {
	ok, err := argon2id.ComparePasswordAndHash(password, hash)
	return err == nil && ok
}

// VerifyDummy — constant-time проверка против заранее посчитанного хеша.
func (h *Argon2Hasher) VerifyDummy(password string) {
	_, _ = argon2id.ComparePasswordAndHash(password, h.dummyHash)
}

// Компиляторная проверка соответствия доменному контракту.
var _ PasswordHasher = (*Argon2Hasher)(nil)
