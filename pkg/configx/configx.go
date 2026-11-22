// Package configx — helpers для 12-factor конфигурации через переменные окружения.
//
// В M2 сервисы переезжают на declarative-конфиг (caarlos0/env);
// до тех пор эти helpers дают единые правила значений по умолчанию.
package configx

import (
	"os"
	"strconv"
	"time"
)

// String возвращает значение переменной окружения key или def, если она пуста.
func String(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// Int возвращает целочисленную переменную окружения key или def при ошибке разбора.
func Int(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// Bool возвращает булеву переменную окружения key или def при ошибке разбора.
func Bool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// Duration возвращает длительность (например "30s", "5m") из key или def.
func Duration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
