// Package http — HTTP-транспорт auth-service: strict-server из контракта
// api/openapi/auth.yaml + проверка JWT + маппинг доменных ошибок в error-envelope.
//
// Правила маппинга:
//
//	domain.ErrEmailTaken         → 409 email_already_exists
//	domain.ErrInvalidCredentials → 401 invalid_credentials
//	domain.ErrAccountLocked      → 423 account_locked (401 там, где контракт не имеет 423)
//	domain.ErrSessionRevoked     → 401 invalid_refresh_token
//	domain.ErrNotFound           → 404 not_found
//	domain.ErrForbidden          → 403 forbidden
//	*service.ValidateError       → 400 validation_failed + details
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

	"github.com/rosterra/platform/pkg/httpx"
	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/services/auth/internal/config"
	"github.com/rosterra/services/auth/internal/domain"
	"github.com/rosterra/services/auth/internal/service"
	gen "github.com/rosterra/services/auth/internal/transport/http/api/gen"
)

// apiDetail — алиас повторяет анонимную структуру details из gen.Error,
// поэтому []apiDetail типо-идентичен полю Details контракта.
type apiDetail = struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// Коды error-envelope.
const (
	codeUnauthorized       = "unauthorized"
	codeForbidden          = "forbidden"
	codeNotFound           = "not_found"
	codeInvalidCredentials = "invalid_credentials"
	codeAccountLocked      = "account_locked"
	codeEmailTaken         = "email_already_exists"
	codeInvalidRefresh     = "invalid_refresh_token"
	codeValidation         = "validation_failed"
	codeInternal           = "internal"
)

// Сообщения безопасны для показа пользователю (не раскрывают internals).
var errMessages = map[string]string{
	codeUnauthorized:       "authentication required",
	codeForbidden:          "insufficient permissions",
	codeNotFound:           "user not found",
	codeInvalidCredentials: "invalid email or password",
	codeAccountLocked:      "account is blocked",
	codeEmailTaken:         "email is already registered",
	codeInvalidRefresh:     "refresh token is invalid or revoked",
	codeValidation:         "request validation failed",
	codeInternal:           "internal server error",
}

type claimsKeyType struct{}

// Server — HTTP-сервер auth-service.
type Server struct {
	httpx    *httpx.Server
	svc      *service.Service
	verifier *jwtx.Verifier
	issuer   *jwtx.Issuer
	log      *slog.Logger
}

// NewServer собирает strict-handler, middleware аутентификации и health-checks.
func NewServer(cfg *config.Config, log *slog.Logger, svc *service.Service, verifier *jwtx.Verifier, issuer *jwtx.Issuer) *Server {
	s := &Server{svc: svc, verifier: verifier, issuer: issuer, log: log}

	router := chi.NewRouter()
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
	s.httpx.SetReady(true) // TODO: переводить в false до готовности PG/Redis/NATS.

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

// authMiddleware разбирает Bearer-токен и кладёт claims в контекст.
// Отсутствующий/невалидный токен не блокирует запрос: необходимость авторизации
// решает конкретный handler (публичные эндпоинты контракта остаются публичными).
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

// errBody собирает error-envelope по коду.
func errBody(code string) gen.Error {
	msg, ok := errMessages[code]
	if !ok {
		msg = code
	}
	return gen.Error{Code: code, Message: msg}
}

// requireAdmin — Claims + error для admin-эндпоинтов.
func (s *Server) requireAdmin(ctx context.Context) (*jwtx.Claims, *gen.Error) {
	c := s.claims(ctx)
	if c == nil {
		return nil, errPtr(codeUnauthorized)
	}
	for _, r := range c.Roles {
		if r == string(domain.RoleAdmin) {
			return c, nil
		}
	}
	return nil, errPtr(codeForbidden)
}

func errPtr(code string) *gen.Error {
	e := errBody(code)
	return &e
}

// validationResp — 400 validation_failed с деталями по полям.
func validationResp(ve *service.ValidateError) gen.Error {
	e := errBody(codeValidation)
	if len(ve.Fields) > 0 {
		details := make([]apiDetail, 0, len(ve.Fields))
		for _, f := range ve.Fields {
			details = append(details, apiDetail{Field: f.Field, Message: f.Message})
		}
		e.Details = &details
	}
	return e
}

// internalErr логирует неожиданную ошибку; HTTP-ответ (500 internal) рендерит
// ResponseErrorHandlerFunc strict-обработчика.
func (s *Server) internalErr(err error, op string) error {
	s.log.Error("internal error", "op", op, "err", err)
	return fmt.Errorf("%s: %w", op, err)
}

func userToGen(u *domain.User) gen.User {
	name := u.Name
	roles := make([]gen.Role, 0, len(u.Roles))
	for _, r := range u.Roles {
		roles = append(roles, gen.Role(r))
	}
	return gen.User{
		Id:            types.UUID(u.ID),
		Email:         types.Email(u.Email),
		Name:          name,
		Status:        gen.UserStatus(u.Status),
		Roles:         roles,
		EmailVerified: u.EmailVerified,
		CreatedAt:     u.CreatedAt,
		UpdatedAt:     u.UpdatedAt,
	}
}

func tokensToGen(sess *service.Session) gen.SessionTokens {
	return gen.SessionTokens{
		AccessToken:      sess.Tokens.AccessToken,
		TokenType:        gen.Bearer,
		ExpiresIn:        int(sess.Tokens.AccessTTL.Seconds()),
		RefreshToken:     sess.Tokens.RefreshToken,
		RefreshExpiresIn: int(sess.Tokens.RefreshTTL.Seconds()),
		User:             userToGen(sess.User),
	}
}

// ---- strict-server: публичные эндпоинты ----

// GetJwks публикует публичные ключи подписи (RFC 7517).
func (s *Server) GetJwks(_ context.Context, _ gen.GetJwksRequestObject) (gen.GetJwksResponseObject, error) {
	return gen.GetJwks200JSONResponse(gen.JsonWebKeySet{Keys: s.issuer.JWKSKeys()}), nil
}

// Register — регистрация + авто-логин.
func (s *Server) Register(ctx context.Context, req gen.RegisterRequestObject) (gen.RegisterResponseObject, error) {
	var ve *service.ValidateError
	sess, err := s.svc.Register(ctx, string(req.Body.Email), req.Body.Password, req.Body.Name)
	switch {
	case err == nil:
		return gen.Register201JSONResponse(tokensToGen(sess)), nil
	case errors.As(err, &ve):
		return gen.Register400JSONResponse(validationResp(ve)), nil
	case errors.Is(err, domain.ErrEmailTaken):
		return gen.Register409JSONResponse(errBody(codeEmailTaken)), nil
	default:
		return nil, s.internalErr(err, "register")
	}
}

// Login — вход по паролю.
func (s *Server) Login(ctx context.Context, req gen.LoginRequestObject) (gen.LoginResponseObject, error) {
	remember := false
	if req.Body.RememberMe != nil {
		remember = *req.Body.RememberMe
	}
	var ve *service.ValidateError
	sess, err := s.svc.Login(ctx, string(req.Body.Email), req.Body.Password, remember)
	switch {
	case err == nil:
		return gen.Login200JSONResponse(tokensToGen(sess)), nil
	case errors.As(err, &ve):
		return gen.Login401JSONResponse(validationResp(ve)), nil
	case errors.Is(err, domain.ErrInvalidCredentials):
		return gen.Login401JSONResponse(errBody(codeInvalidCredentials)), nil
	case errors.Is(err, domain.ErrAccountLocked):
		return gen.Login423JSONResponse(errBody(codeAccountLocked)), nil
	default:
		return nil, s.internalErr(err, "login")
	}
}

// Refresh — ротация refresh-токена.
func (s *Server) Refresh(ctx context.Context, req gen.RefreshRequestObject) (gen.RefreshResponseObject, error) {
	sess, err := s.svc.Refresh(ctx, req.Body.RefreshToken)
	switch {
	case err == nil:
		return gen.Refresh200JSONResponse(tokensToGen(sess)), nil
	case errors.Is(err, domain.ErrAccountLocked):
		return gen.Refresh401JSONResponse(errBody(codeAccountLocked)), nil
	case errors.Is(err, domain.ErrInvalidRefreshToken), errors.Is(err, domain.ErrSessionRevoked):
		return gen.Refresh401JSONResponse(errBody(codeInvalidRefresh)), nil
	default:
		return nil, s.internalErr(err, "refresh")
	}
}

// Logout — отзыв refresh-сессии.
func (s *Server) Logout(ctx context.Context, req gen.LogoutRequestObject) (gen.LogoutResponseObject, error) {
	if s.claims(ctx) == nil {
		return gen.Logout401JSONResponse(errBody(codeUnauthorized)), nil
	}
	if err := s.svc.Logout(ctx, req.Body.RefreshToken); err != nil {
		return nil, s.internalErr(err, "logout")
	}
	return gen.Logout204Response{}, nil
}

// GetMe — текущий пользователь.
func (s *Server) GetMe(ctx context.Context, _ gen.GetMeRequestObject) (gen.GetMeResponseObject, error) {
	claims := s.claims(ctx)
	if claims == nil {
		return gen.GetMe401JSONResponse(errBody(codeUnauthorized)), nil
	}
	user, err := s.svc.Me(ctx, claims.UserID)
	switch {
	case err == nil:
		return gen.GetMe200JSONResponse(userToGen(user)), nil
	case errors.Is(err, domain.ErrNotFound):
		return gen.GetMe401JSONResponse(errBody(codeUnauthorized)), nil
	case errors.Is(err, domain.ErrAccountLocked):
		return gen.GetMe401JSONResponse(errBody(codeAccountLocked)), nil
	default:
		return nil, s.internalErr(err, "get me")
	}
}

// ---- strict-server: admin-эндпоинты (role=admin) ----

// AdminListUsers — список с пагинацией/поиском/фильтрами.
func (s *Server) AdminListUsers(ctx context.Context, req gen.AdminListUsersRequestObject) (gen.AdminListUsersResponseObject, error) {
	if _, e := s.requireAdmin(ctx); e != nil {
		if e.Code == codeUnauthorized {
			return gen.AdminListUsers401JSONResponse(*e), nil
		}
		return gen.AdminListUsers403JSONResponse(*e), nil
	}

	p := req.Params
	users, total, err := s.svc.ListUsers(ctx, domain.ListFilter{
		Query:   p.Query,
		Status:  mapStatus(p.Status),
		Role:    mapRole(p.Role),
		Page:    derefInt(p.Page),
		PerPage: derefInt(p.PerPage),
	})
	if err != nil {
		return nil, s.internalErr(err, "admin list users")
	}
	items := make([]gen.User, 0, len(users))
	for i := range users {
		items = append(items, userToGen(&users[i]))
	}
	return gen.AdminListUsers200JSONResponse(gen.UserPage{
		Items:   items,
		Total:   total,
		Page:    derefInt(p.Page),
		PerPage: derefInt(p.PerPage),
	}), nil
}

// AdminGetUser — пользователь по id.
func (s *Server) AdminGetUser(ctx context.Context, req gen.AdminGetUserRequestObject) (gen.AdminGetUserResponseObject, error) {
	if _, e := s.requireAdmin(ctx); e != nil {
		return adminErrResp(e), nil
	}
	user, err := s.svc.GetUser(ctx, uuid.UUID(req.Id))
	if errors.Is(err, domain.ErrNotFound) {
		return gen.AdminGetUser404JSONResponse(errBody(codeNotFound)), nil
	}
	if err != nil {
		return nil, s.internalErr(err, "admin get user")
	}
	return gen.AdminGetUser200JSONResponse(userToGen(user)), nil
}

// AdminSetUserStatus — блокировка/разблокировка (с отзывом сессий при блоке).
func (s *Server) AdminSetUserStatus(ctx context.Context, req gen.AdminSetUserStatusRequestObject) (gen.AdminSetUserStatusResponseObject, error) {
	if _, e := s.requireAdmin(ctx); e != nil {
		return adminSetStatusErrResp(e), nil
	}
	var ve *service.ValidateError
	res, err := s.svc.SetStatus(ctx, uuid.UUID(req.Id), domain.UserStatus(req.Body.Status), req.Body.Reason)
	switch {
	case err == nil:
		return gen.AdminSetUserStatus200JSONResponse(userToGen(res.User)), nil
	case errors.As(err, &ve):
		return gen.AdminSetUserStatus400JSONResponse(validationResp(ve)), nil
	case errors.Is(err, domain.ErrNotFound):
		return gen.AdminSetUserStatus404JSONResponse(errBody(codeNotFound)), nil
	default:
		return nil, s.internalErr(err, "admin set status")
	}
}

// AdminSetUserRoles — замена набора ролей.
func (s *Server) AdminSetUserRoles(ctx context.Context, req gen.AdminSetUserRolesRequestObject) (gen.AdminSetUserRolesResponseObject, error) {
	if _, e := s.requireAdmin(ctx); e != nil {
		return adminSetRolesErrResp(e), nil
	}
	roles := make([]domain.Role, 0, len(req.Body.Roles))
	for _, r := range req.Body.Roles {
		roles = append(roles, domain.Role(r))
	}
	var ve *service.ValidateError
	user, err := s.svc.SetRoles(ctx, uuid.UUID(req.Id), roles)
	switch {
	case err == nil:
		return gen.AdminSetUserRoles200JSONResponse(userToGen(user)), nil
	case errors.As(err, &ve):
		return gen.AdminSetUserRoles400JSONResponse(validationResp(ve)), nil
	case errors.Is(err, domain.ErrNotFound):
		return gen.AdminSetUserRoles404JSONResponse(errBody(codeNotFound)), nil
	default:
		return nil, s.internalErr(err, "admin set roles")
	}
}

// ---- маппинг-хелперы параметров ----

func mapStatus(s *gen.UserStatus) *domain.UserStatus {
	if s == nil {
		return nil
	}
	v := domain.UserStatus(*s)
	return &v
}

func mapRole(r *gen.Role) *domain.Role {
	if r == nil {
		return nil
	}
	v := domain.Role(*r)
	return &v
}

func derefInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func adminErrResp(e *gen.Error) gen.AdminGetUserResponseObject {
	switch e.Code {
	case codeUnauthorized:
		return gen.AdminGetUser401JSONResponse(*e)
	case codeForbidden:
		return gen.AdminGetUser403JSONResponse(*e)
	default:
		return gen.AdminGetUser404JSONResponse(*e)
	}
}

func adminSetStatusErrResp(e *gen.Error) gen.AdminSetUserStatusResponseObject {
	switch e.Code {
	case codeUnauthorized:
		return gen.AdminSetUserStatus401JSONResponse(*e)
	case codeForbidden:
		return gen.AdminSetUserStatus403JSONResponse(*e)
	default:
		return gen.AdminSetUserStatus404JSONResponse(*e)
	}
}

func adminSetRolesErrResp(e *gen.Error) gen.AdminSetUserRolesResponseObject {
	switch e.Code {
	case codeUnauthorized:
		return gen.AdminSetUserRoles401JSONResponse(*e)
	case codeForbidden:
		return gen.AdminSetUserRoles403JSONResponse(*e)
	default:
		return gen.AdminSetUserRoles404JSONResponse(*e)
	}
}
