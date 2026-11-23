// Command users — точка входа users-service: профили пользователей.
//
// Сборка через uber fx (зеркало auth): PG (+миграции), NATS, jwtx-verifier,
// репозитории, сервис, HTTP strict-server (users.yaml), gRPC (users.v1),
// outbox-релей (svc.users.>) и consumer событий auth (user.created → профиль).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	"github.com/rosterra/platform/pkg/jwtx"
	"github.com/rosterra/platform/pkg/logger"
	"github.com/rosterra/platform/pkg/otelx"
	"go.uber.org/fx"
	"google.golang.org/grpc"

	usersv1 "github.com/rosterra/services/users/gen/users/v1"
	"github.com/rosterra/services/users/internal/config"
	"github.com/rosterra/services/users/internal/domain"
	"github.com/rosterra/services/users/internal/repo/postgres"
	"github.com/rosterra/services/users/internal/service"
	appgrpc "github.com/rosterra/services/users/internal/transport/grpcusers"
	apphttp "github.com/rosterra/services/users/internal/transport/http"
	"github.com/rosterra/services/users/internal/transport/mq"
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
			newOtel,
			newPool,
			newNATS,
			newJetStream,
			newVerifier,
			fx.Annotate(postgres.NewProfileRepo, fx.As(new(domain.ProfileRepo))),
			fx.Annotate(postgres.NewOutboxRepo, fx.As(new(domain.OutboxRepo))),
			fx.Annotate(postgres.NewEventProcessedRepo, fx.As(new(domain.EventProcessedRepo))),
			newRelay,
			service.New,
			apphttp.NewServer,
			appgrpc.NewServer,
			newGRPCListen,
			mq.NewConsumer,
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

// newOtel инициализирует трассировку (OTLP → Jaeger); без OTEL_ENDPOINT — noop.
func newOtel(cfg *config.Config) (*otelx.OTel, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return otelx.Setup(ctx, otelx.Config{
		ServiceName:  "rosterra/users",
		OTLPEndpoint: cfg.OTELEndpoint,
	})
}

// newPool открывает пул и поднимает миграции до создания репозиториев.
func newPool(cfg *config.Config, log *slog.Logger) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return postgres.NewPool(ctx, cfg.PostgresDSN, log)
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

// newVerifier проверяет access-JWT (тот же издатель rosterra-auth, что и в auth-service).
// В dev оба сервиса разделяют JWT_SEED; в prod — общий ключ/доверенный JWKS.
func newVerifier(cfg *config.Config) (*jwtx.Verifier, error) {
	return jwtx.NewVerifier(cfg.JWTSeed, "rosterra-auth", "rosterra")
}

func newRelay(cfg *config.Config, ob domain.OutboxRepo, js nats.JetStreamContext, log *slog.Logger, otelInst *otelx.OTel) *mq.Relay {
	return mq.NewRelay(ob, js, cfg.RelayInterval, log, otelInst)
}

// newGRPCListen — слушатель gRPC-порта.
func newGRPCListen(cfg *config.Config) (net.Listener, error) {
	return net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
}

// registerLifecycle — старт/стоп HTTP, gRPC, релея и consumer'а.
func registerLifecycle(
	lc fx.Lifecycle,
	srv *apphttp.Server,
	grpcSrv *appgrpc.Server,
	lis net.Listener,
	relay *mq.Relay,
	consumer *mq.Consumer,
	natsc *NATSConnection,
	otelInst *otelx.OTel,
	log *slog.Logger,
) {
	appCtx, cancel := context.WithCancel(context.Background())
	grpcAPI := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			otelx.GRPCUnaryServerInterceptor("rosterra/users"),
		),
	)
	usersv1.RegisterUsersServiceServer(grpcAPI, grpcSrv)

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			go func() {
				if err := srv.Run(appCtx); err != nil {
					log.Error("http server", "err", err)
				}
			}()
			go func() {
				if err := grpcAPI.Serve(lis); err != nil {
					log.Error("grpc server", "err", err)
				}
			}()
			go relay.Run(appCtx)
			go consumer.Run(appCtx)
			log.Info("users service started", "grpc_port", lis.Addr().String())
			return nil
		},
		OnStop: func(ctx context.Context) error {
			cancel()
			consumer.Stop()
			relay.Stop()
			if err := srv.Shutdown(ctx); err != nil {
				return err
			}
			grpcAPI.GracefulStop()
			natsc.Conn.Drain()
			if err := otelInst.Shutdown(ctx); err != nil {
				log.Warn("otel shutdown", "err", err)
			}
			log.Info("users service stopped")
			return nil
		},
	})
}
