// Package config — конфигурация users-service из переменных окружения.
package config

import (
	"time"

	"github.com/rosterra/platform/pkg/configx"
)

// Config — все настройки сервиса.
type Config struct {
	// Env — окружение: "dev", "stage", "prod".
	Env string
	// ServiceName — имя сервиса для логов и метрик.
	ServiceName string
	// LogLevel — уровень логирования (debug|info|warn|error).
	LogLevel string
	// HTTPPort — порт HTTP-сервера.
	HTTPPort int
	// GRPCPort — порт gRPC-сервера (межсервисные вызовы).
	GRPCPort int

	// PostgresDSN — строка подключения к PostgreSQL (профили; dev — та же БД, что auth).
	PostgresDSN string
	// NATSURL — адрес брокера (события user.created/profile.updated).
	NATSURL string
	// JWTSeed — seed ключа Ed25519 издателя токенов (общий с auth-service).
	// Пусто → эфемерный верификатор не имеет смысла; в этом случае HTTP-авторизация
	// будет отклонять токены (WARN).
	JWTSeed string
	// RelayInterval — интервал опроса outbox-релея.
	RelayInterval time.Duration
}

// Load читает конфигурацию из окружения, подставляя безопасные значения по умолчанию.
func Load() *Config {
	return &Config{
		Env:         configx.String("ENV", "dev"),
		ServiceName: configx.String("SERVICE_NAME", "users"),
		LogLevel:    configx.String("LOG_LEVEL", "info"),
		HTTPPort:    configx.Int("HTTP_PORT", 8082),
		GRPCPort:    configx.Int("GRPC_PORT", 9082),

		PostgresDSN: configx.String("POSTGRES_DSN",
			"postgres://rosterra:rosterra@localhost:5432/rosterra?sslmode=disable"),
		NATSURL:       configx.String("NATS_URL", "nats://localhost:4222"),
		JWTSeed:       configx.String("JWT_SEED", ""),
		RelayInterval: configx.Duration("RELAY_INTERVAL", 1*time.Second),
	}
}
