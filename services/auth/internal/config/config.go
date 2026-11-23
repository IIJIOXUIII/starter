// Package config — конфигурация auth-service из переменных окружения.
package config

import (
	"time"

	"github.com/rosterra/platform/pkg/configx"
)

// Config — все настройки auth-service.
type Config struct {
	// Env — окружение: "dev", "stage", "prod".
	Env string
	// ServiceName — имя сервиса для логов и метрик.
	ServiceName string
	// LogLevel — уровень логирования (debug|info|warn|error).
	LogLevel string
	// HTTPPort — порт HTTP-сервера.
	HTTPPort int

	// PostgresDSN — строка подключения к PostgreSQL (схема auth).
	PostgresDSN string
	// RedisAddr — адрес Redis (refresh-сессии).
	RedisAddr string
	// NATSURL — адрес брокера (события).
	NATSURL string

	// JWTSeed — seed приватного ключа Ed25519 (hex/base64, 64 байта).
	// Пусто → эфемерный ключ с WARN-логом (только dev: JWKS меняется между рестартами).
	JWTSeed string
	// AccessTTL — время жизни access-токена.
	AccessTTL time.Duration
	// RefreshTTL — TTL refresh-сессии при remember_me=false.
	RefreshTTL time.Duration
	// RefreshTTLLong — TTL refresh-сессии при remember_me=true.
	RefreshTTLLong time.Duration
	// RelayInterval — интервал опроса outbox-релея.
	RelayInterval time.Duration
	// OTELEndpoint — базовый URL OTLP HTTP-коллектора (Jaeger).
	// Пусто → трассировка выключена (noop).
	OTELEndpoint string
}

// Load читает конфигурацию из окружения, подставляя безопасные значения по умолчанию.
func Load() *Config {
	return &Config{
		Env:         configx.String("ENV", "dev"),
		ServiceName: configx.String("SERVICE_NAME", "auth"),
		LogLevel:    configx.String("LOG_LEVEL", "info"),
		HTTPPort:    configx.Int("HTTP_PORT", 8081),

		PostgresDSN: configx.String("POSTGRES_DSN",
			"postgres://rosterra:rosterra@localhost:5432/rosterra?sslmode=disable"),
		RedisAddr: configx.String("REDIS_ADDR", "localhost:6379"),
		NATSURL:   configx.String("NATS_URL", "nats://localhost:4222"),

		JWTSeed:        configx.String("JWT_SEED", ""),
		AccessTTL:      configx.Duration("ACCESS_TTL", 15*time.Minute),
		RefreshTTL:     configx.Duration("REFRESH_TTL", 12*time.Hour),
		RefreshTTLLong: configx.Duration("REFRESH_TTL_LONG", 30*24*time.Hour),
		RelayInterval:  configx.Duration("RELAY_INTERVAL", 1*time.Second),
		OTELEndpoint:   configx.String("OTEL_ENDPOINT", ""),
	}
}
