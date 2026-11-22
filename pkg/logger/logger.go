// Package logger — общий structured-логгер платформы (slog, JSON).
//
// Все сервисы обязаны получать логгер только через этот пакет:
// единый формат записей позволяет Loki/Grafana парсить логи без адаптеров.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// New создаёт JSON-логгер с постоянным атрибутом "service".
//
// level — "debug", "info", "warn" или "error" (иное значение → info).
func New(level, service string) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: parseLevel(level),
	})).With("service", service)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
