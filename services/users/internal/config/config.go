// Package config — конфигурация users-service из переменных окружения.
package config

import (
	"github.com/rosterra/platform/pkg/configx"
)

// Config — все настройки сервиса. Дополнения в M5: DSN PostgreSQL, NATS.
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
}

// Load читает конфигурацию из окружения, подставляя безопасные значения по умолчанию.
func Load() *Config {
	return &Config{
		Env:         configx.String("ENV", "dev"),
		ServiceName: configx.String("SERVICE_NAME", "users"),
		LogLevel:    configx.String("LOG_LEVEL", "info"),
		HTTPPort:    configx.Int("HTTP_PORT", 8082),
		GRPCPort:    configx.Int("GRPC_PORT", 9082),
	}
}
