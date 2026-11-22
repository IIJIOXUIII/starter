// Command auth — точка входа auth-service: регистрация, JWT, сессии, роли.
//
// Сборка через uber fx: провайдеры инфраструктуры (PG/Redis/NATS),
// домен-репозитории, бизнес-сервис и HTTP-транспорт; lifecycle —
// graceful-запуск и остановка сервера и outbox-релея.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"

	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/platform/pkg/logger"
	"github.com/rosterra/services/auth/internal/config"
	"github.com/rosterra/services/auth/internal/domain"
	"github.com/rosterra/services/auth/internal/repo/postgres"
	redisrepo "github.com/rosterra/services/auth/internal/repo/redis"
	"github.com/rosterra/services/auth/internal/service"
	apphttp "github.com/rosterra/services/auth/internal/transport/http"
	"github.com/rosterra/services/auth/internal/transport/mq"
)

// NATSConnection — подключение + JetStream-контекст одним значением.
type NATSConnection struct {
	Conn *nats.Conn
	JS   nats.JetStreamContext
}

func main() {
	app := fx.New(
		fx.Provide(
			config.Load,
			newLogger,
			newPool,
			newRedis,
			newNATS,
			newJetStream,
			newIssuer,
			newTokenIssuer,
			newPasswordHasher,
			fx.Annotate(postgres.NewUserRepo, fx.As(new(domain.UserRepo))),
			fx.Annotate(postgres.NewOutboxRepo, fx.As(new(domain.OutboxRepo))),
			fx.Annotate(redisrepo.NewSessionRepo, fx.As(new(domain.SessionRepo))),
			newRelay,
			newServiceConfig,
			service.New,
			newVerifier,
			apphttp.NewServer,
		),
		fx.Invoke(registerLifecycle),
	)

	if err := app.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: fx build: %v\n", err)
		os.Exit(1)
	}

	// fx сам перехватывает SIGINT/SIGTERM: запускает OnStart, ждёт сигнал, зовёт OnStop.
	app.Run()
}

func newLogger(cfg *config.Config) *slog.Logger {
	return logger.New(cfg.LogLevel, cfg.ServiceName)
}

// newPool открывает пул и поднимает миграции до создания репозиториев.
// (Миграции синхронны в конструкторе: fx гарантирует порядок построения графа.)
func newPool(cfg *config.Config, log *slog.Logger) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return postgres.NewPool(ctx, cfg.PostgresDSN, log)
}

func newPasswordHasher() (service.PasswordHasher, error) {
	return service.NewArgon2Hasher()
}

func newRedis(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
}

func newNATS(cfg *config.Config) (*NATSConnection, error) {
	conn, err := nats.Connect(cfg.NATSURL,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
		nats.Name(cfg.ServiceName))
	if err != nil {
		return nil, fmt.Errorf("nats connect: %w", err)
	}
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("jetstream: %w", err)
	}
	return &NATSConnection{Conn: conn, JS: js}, nil
}

// newJetStream — fx-провайдер интерфейса JetStream (из NATSConnection).
func newJetStream(nc *NATSConnection) nats.JetStreamContext {
	return nc.JS
}

// newIssuer создаёт выпускающий ключ; без JWT_SEED — эфемерный (WARN).
func newIssuer(cfg *config.Config, log *slog.Logger) (*jwtx.Issuer, error) {
	issuer, ephemeral, err := jwtx.NewIssuer(cfg.JWTSeed, "rosterra-auth", "rosterra", cfg.AccessTTL)
	if err != nil {
		return nil, err
	}
	if ephemeral {
		log.Warn("JWT_SEED не задан — используется эфемерный ключ: JWKS меняется после рестарта, " +
			"все access-токены недействительны. Задайте JWT_SEED для dev-стабильности.")
	}
	return issuer, nil
}

func newVerifier(cfg *config.Config) (*jwtx.Verifier, error) {
	return jwtx.NewVerifier(cfg.JWTSeed, "rosterra-auth", "rosterra")
}

// newTokenIssuer адаптирует *jwtx.Issuer к service.TokenIssuer.
func newTokenIssuer(issuer *jwtx.Issuer) service.TokenIssuer {
	return &jwtIssuerAdapter{issuer: issuer}
}

type jwtIssuerAdapter struct {
	issuer *jwtx.Issuer
}

// Issue выпускает access-токен.
func (a *jwtIssuerAdapter) Issue(userID, sessionID uuid.UUID, email string, roles []string) (string, time.Duration, error) {
	return a.issuer.Issue(userID, sessionID, email, roles)
}

func newRelay(cfg *config.Config, ob domain.OutboxRepo, js nats.JetStreamContext, log *slog.Logger) *mq.Relay {
	return mq.NewRelay(ob, js, cfg.RelayInterval, log)
}
func newServiceConfig(cfg *config.Config) service.Config {
	return service.Config{
		AccessTTL:      cfg.AccessTTL,
		RefreshTTL:     cfg.RefreshTTL,
		RefreshTTLLong: cfg.RefreshTTLLong,
	}
}

// registerLifecycle — старт/стоп HTTP-сервера и релея.
func registerLifecycle(lc fx.Lifecycle, srv *apphttp.Server, relay *mq.Relay, natsc *NATSConnection, log *slog.Logger) {
	appCtx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := srv.Run(appCtx); err != nil {
					log.Error("http server", "err", err)
				}
			}()
			go relay.Run(appCtx)
			log.Info("auth service started")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			relay.Stop()
			if err := srv.Shutdown(ctx); err != nil {
				return err
			}
			natsc.Conn.Drain()
			log.Info("auth service stopped")
			return nil
		},
	})
}
