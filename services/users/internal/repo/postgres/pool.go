// Package postgres — реализации портов users-service на pgx.
package postgres

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/rosterra/services/users/migrations"
)

// NewPool открывает пул соединений и поднимает миграции (embedded goose).
// Миграции выполняются синхронно до возврата: зависимые провайдеры
// (репозитории) получают уже готовую схему.
func NewPool(ctx context.Context, dsn string, log *slog.Logger) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	sqlDB := stdlib.OpenDBFromPool(pool)
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		sqlDB.Close()
		pool.Close()
		return nil, fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.UpContext(ctx, sqlDB, "."); err != nil {
		sqlDB.Close()
		pool.Close()
		return nil, fmt.Errorf("goose up: %w", err)
	}
	log.Info("postgres migrations applied")
	return pool, nil
}

// ClosePool — graceful-закрытие пула (fx lifecycle).
func ClosePool(pool *pgxpool.Pool) error {
	pool.Close()
	return nil
}
