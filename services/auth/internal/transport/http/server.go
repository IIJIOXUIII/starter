// Package http — HTTP-транспорт auth-service.
//
// В M2 сюда генерируется strict-server из api/openapi/auth.yaml (oapi-codegen);
// сейчас — каркас: health/readiness и ping для проверки связки compose.
package http

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/rosterra/platform/pkg/httpx"
	"github.com/rosterra/services/auth/internal/config"
)

// Server — HTTP-сервер сервиса.
type Server struct {
	httpx *httpx.Server
}

// NewServer собирает маршруты, health-эндпоинты и общий httpx-каркас.
func NewServer(cfg *config.Config, log *slog.Logger) *Server {
	mux := http.NewServeMux()

	// Плейсхолдер доменного API; заменяется strict-server'ом из контракта в M2.
	mux.HandleFunc("GET /v1/ping", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"service":"auth"}`)
	})

	srv := httpx.New(fmt.Sprintf(":%d", cfg.HTTPPort), mux, log)
	srv.SetReady(true) // В M2 — только после успешной инициализации PG/Redis/NATS.

	return &Server{httpx: srv}
}

// Run запускает сервер (блокируется до остановки контекста).
func (s *Server) Run(ctx context.Context) error {
	return s.httpx.Run(ctx)
}

// Shutdown корректно останавливает сервер с дренажом соединений.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpx.Shutdown(ctx)
}
