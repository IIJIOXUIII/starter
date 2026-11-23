// Package http — HTTP-транспорт users-service: strict-server из контракта
// api/openapi/users.yaml + проверка JWT (тот же издатель, что в auth-service).
//
// Правила доступа:
//
//	ListProfiles   → только admin
//	GetProfile     → владелец или admin
//	UpdateProfile  → владелец или admin; notes — только admin
//
// Маппинг ошибок: domain.ErrNotFound → 404, ErrForbidden/ErrUnauthorized → 403/401,
// *service.ValidateError → 400.
package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/oapi-codegen/runtime/types"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/rosterra/platform/pkg/httpx"
	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/platform/pkg/otelx"
	"github.com/rosterra/services/users/internal/config"
	"github.com/rosterra/services/users/internal/domain"
	"github.com/rosterra/services/users/internal/service"
	gen "github.com/rosterra/services/users/internal/transport/http/api/gen"
)

// Коды error-envelope.
const (
	codeUnauthorized = "unauthorized"
	codeForbidden    = "forbidden"
	codeNotFound     = "not_found"
	codeValidation   = "validation_failed"
	codeInternal     = "internal"
)

// Сообщения безопасны для показа.
var errMessages = map[string]string{
	codeUnauthorized: "authentication required",
	codeForbidden:    "insufficient permissions",
	codeNotFound:     "profile not found",
	codeValidation:   "request validation failed",
	codeInternal:     "internal server error",
}

type claimsKeyType struct{}

// Server — HTTP-сервер users-service.
type Server struct {
	httpx    *httpx.Server
	svc      *service.Service
	verifier *jwtx.Verifier
	log      *slog.Logger
}

// NewServer собирает strict-handler, middleware аутентификации и health-checks.
// Зависимость от otel (не используется напрямую) фиксирует порядок fx-конструирования:
// глобальный TracerProvider должен быть установлен до создания middleware.
func NewServer(cfg *config.Config, log *slog.Logger, svc *service.Service, verifier *jwtx.Verifier, _ *otelx.OTel) *Server {
	s := &Server{svc: svc, verifier: verifier, log: log}

	router := chi.NewRouter()

	// Наблюдаемость: метрики (route-лейбл из chi) + серверные спаны.
	// (Все middleware — строго до регистрации маршрутов.)
	router.Use(otelx.MetricsMiddleware(cfg.ServiceName))
	router.Use(otelx.TracingMiddleware(cfg.ServiceName))
	router.Get("/metrics", promhttp.Handler().ServeHTTP)

	registerHealthChi(router, func() bool { return true })

	strict := gen.NewStrictHandlerWithOptions(s, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Warn("bad request", "err", err, "path", r.URL.Path)
			httpx.JSON(w, http.StatusBadRequest, errBody(codeValidation))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error("internal api error", "err", err, "path", r.URL.Path)
			httpx.JSON(w, http.StatusInternalServerError, errBody(codeInternal))
		},
	})

	gen.HandlerWithOptions(strict, gen.ChiServerOptions{
		BaseRouter:  router,
		Middlewares: []gen.MiddlewareFunc{s.authMiddleware()},
	})

	s.httpx = httpx.New(fmt.Sprintf(":%d", cfg.HTTPPort), router, log)
	s.httpx.SetReady(true) // TODO(M6): false до готовности PG/NATS.

	return s
}

// Run запускает сервер (блокируется до остановки контекста).
func (s *Server) Run(ctx context.Context) error { return s.httpx.Run(ctx) }

// Shutdown корректно останавливает сервер с дренажом соединений.
func (s *Server) Shutdown(ctx context.Context) error { return s.httpx.Shutdown(ctx) }

// registerHealthChi — health/readiness на chi-роутере.
func registerHealthChi(r chi.Router, ready func() bool) {
	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if ready() {
			httpx.JSON(w, http.StatusOK, map[string]string{"status": "ready"})
			return
		}
		httpx.JSON(w, http.StatusServiceUnavailable, map[string]string{"status": "not_ready"})
	})
}

// authMiddleware разбирает Bearer-токен (тот же издатель, что в auth) и кладёт
// claims в контекст. Невалидный токен не блокирует: авторизацию решает handler.
func (s *Server) authMiddleware() gen.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			if strings.HasPrefix(authz, "Bearer ") {
				token := strings.TrimPrefix(authz, "Bearer ")
				if claims, err := s.verifier.Verify(token); err == nil {
					r = r.WithContext(context.WithValue(r.Context(), claimsKeyType{}, claims))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) claims(ctx context.Context) *jwtx.Claims {
	c, _ := ctx.Value(claimsKeyType{}).(*jwtx.Claims)
	return c
}

// isAdmin проверяет наличие роли admin в claims.
func isAdmin(c *jwtx.Claims) bool {
	if c == nil {
		return false
	}
	for _, r := range c.Roles {
		if r == "admin" {
			return true
		}
	}
	return false
}

// errBody собирает error-envelope по коду.
func errBody(code string) gen.Error {
	msg, ok := errMessages[code]
	if !ok {
		msg = code
	}
	return gen.Error{Code: code, Message: msg}
}

// domainError маппит доменную ошибку в *gen.Error (строгий ответ non-200 вернёт strict-обёртка).
func domainError(err error) *gen.Error {
	switch {
	case errors.Is(err, domain.ErrUnauthorized):
		e := errBody(codeUnauthorized)
		return &e
	case errors.Is(err, domain.ErrForbidden):
		e := errBody(codeForbidden)
		return &e
	case errors.Is(err, domain.ErrNotFound):
		e := errBody(codeNotFound)
		return &e
	default:
		return nil
	}
}

// validationError возвращает 400-ошибку контракта с деталями по полям.
func validationError(ve *service.ValidateError) *gen.Error {
	e := errBody(codeValidation)
	if len(ve.Fields) > 0 {
		details := make([]struct {
			Field   string `json:"field"`
			Message string `json:"message"`
		}, 0, len(ve.Fields))
		for _, f := range ve.Fields {
			details = append(details, struct {
				Field   string `json:"field"`
				Message string `json:"message"`
			}{Field: f.Field, Message: f.Message})
		}
		e.Details = &details
	}
	return &e
}

// profileToGen — домен → контракт.
func profileToGen(p *domain.Profile) gen.Profile {
	avatar := p.AvatarURL
	notes := p.Notes
	if avatar != nil && *avatar == "" {
		avatar = nil
	}
	return gen.Profile{
		UserId:      types.UUID(p.UserID),
		DisplayName: p.DisplayName,
		AvatarUrl:   avatar,
		Notes:       notes,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func parseUUIDs(raw string) ([]uuid.UUID, *gen.Error) {
	parts := strings.Split(raw, ",")
	if len(parts) > 100 {
		return nil, errPtr(codeValidation)
	}
	ids := make([]uuid.UUID, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := uuid.Parse(part)
		if err != nil {
			return nil, errPtr(codeValidation)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func errPtr(code string) *gen.Error {
	e := errBody(code)
	return &e
}
