// Package config — конфигурация auth-service из переменных окружения.
package config

import (
	"github.com/rosterra/platform/pkg/configx"
)

// Config — все настройки сервиса. Дополнения в M2: DSN PostgreSQL, Redis,
// NATS, параметры JWT и Argon2id.
type Config struct {
	// Env — окружение: "dev", "stage", "prod".
	Env string
	// ServiceName — имя сервиса для логов и метрик.
	ServiceName string
	// LogLevel — уровень логирования (debug|info|warn|error).
	LogLevel string
	// HTTPPort — порт HTTP-сервера.
	HTTPPort int
}

// Load читает конфигурацию из окружения, подставляя безопасные значения по умолчанию.
func Load() *Config {
	return &Config{
		Env:         configx.String("ENV", "dev"),
		ServiceName: configx.String("SERVICE_NAME", "auth"),
		LogLevel:    configx.String("LOG_LEVEL", "info"),
		HTTPPort:    configx.Int("HTTP_PORT", 8081),
	}
}
