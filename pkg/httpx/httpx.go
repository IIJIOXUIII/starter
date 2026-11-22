// Package httpx — общий HTTP-каркас сервисов: сервер с таймаутами,
// health/readiness-эндпоинты и middleware логирования запросов.
//
// Health-эндпоинты используются Docker Compose, Kubernetes (liveness/readiness)
// и локальными хелсчеками:
//
//	GET /healthz — процесс жив (всегда 200 у запущенного сервера);
//	GET /readyz  — готов принимать трафик (200/503 в зависимости от SetReady).
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"
)

// Server — обёртка над http.Server с безопасными таймаутами и readiness-флагом.
type Server struct {
	http  *http.Server
	ready atomic.Bool
}

// New создаёт сервер на базе переданного handler'а (chi-роутер или http.ServeMux).
// Для *http.ServeMux health/readiness регистрируются автоматически;
// для остальных — используйте RegisterHealth или httpx.JSON вручную.
func New(addr string, handler http.Handler, log *slog.Logger) *Server {
	s := &Server{}
	if mux, ok := handler.(*http.ServeMux); ok {
		RegisterHealth(mux, s.Ready)
	}
	s.http = &http.Server{
		Addr:              addr,
		Handler:           LogRequests(log)(handler),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		ErrorLog:          slog.NewLogLogger(log.Handler(), slog.LevelError),
	}
	return s
}

// SetReady управляет readiness-флагом (вызывать после инициализации БД и т.п.).
func (s *Server) SetReady(v bool) { s.ready.Store(v) }

// Ready возвращает текущее значение readiness-флага.
func (s *Server) Ready() bool { return s.ready.Load() }

// Run запускает сервер и блокируется до остановки контекста или ошибки.
// http.ErrServerClosed преобразуется в nil.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)
	go func() { errCh <- s.http.ListenAndServe() }()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		return nil
	}
}

// Shutdown корректно останавливает сервер с дренажом активных соединений.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.http.Shutdown(ctx)
}

// RegisterHealth добавляет /healthz и /readyz на переданный mux.
// ready может быть nil — тогда readiness всегда true.
func RegisterHealth(mux *http.ServeMux, ready func() bool) {
	if ready == nil {
		ready = func() bool { return true }
	}
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready() {
			JSON(w, http.StatusOK, map[string]string{"status": "ready"})
			return
		}
		JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	})
}

// JSON — единый способ ответа JSON для сервисов платформы.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body) //nolint:errcheck // клиент мог отключиться — это не ошибка сервиса
}

// LogRequests — middleware структурного логирования: метод, путь, статус, длительность.
func LogRequests(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", rec.status,
				"duration", time.Since(start).String(),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}
