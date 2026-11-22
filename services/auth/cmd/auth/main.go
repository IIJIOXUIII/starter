// Command auth — точка входа auth-service: регистрация, JWT, сессии, роли.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rosterra/platform/pkg/logger"
	"github.com/rosterra/services/auth/internal/config"
	http "github.com/rosterra/services/auth/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel, cfg.ServiceName)
	log.Info("starting service", "env", cfg.Env, "port", cfg.HTTPPort)

	srv := http.NewServer(cfg, log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() { errCh <- srv.Run(ctx) }()

	select {
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("server: %w", err)
		}
	case <-ctx.Done():
		log.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		if !errors.Is(err, context.DeadlineExceeded) {
			return fmt.Errorf("shutdown: %w", err)
		}
	}

	log.Info("service stopped")
	return nil
}
