package otelx

// gRPC unary-интерцепторы: серверные/клиентские спаны и пропагация W3C
// через metadata. Ручная реализация вместо deprecated otelgrpc —
// чтобы не зависеть от смены рекомендаций contrib-репозитория.

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// metadataCarrier адаптирует grpc metadata.MD к TextMapCarrier.
type metadataCarrier metadata.MD

func (m metadataCarrier) Get(key string) string {
	values := metadata.MD(m).Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func (m metadataCarrier) Set(key, value string) {
	metadata.MD(m).Set(key, value)
}

func (m metadataCarrier) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GRPCUnaryServerInterceptor — спан сервера, контекст из входящего metadata.
func GRPCUnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	tracer := otel.Tracer(serviceName)

	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		md, _ := metadata.FromIncomingContext(ctx)
		ctx = Propagator.Extract(ctx, metadataCarrier(md))

		spanCtx, span := tracer.Start(ctx, fmt.Sprintf("grpc.%s", info.FullMethod),
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", info.FullMethod)),
		)
		defer span.End()

		resp, err := handler(spanCtx, req)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return resp, err
	}
}

// GRPCUnaryClientInterceptor — спан клиента, контекст в исходящий metadata.
func GRPCUnaryClientInterceptor(serviceName string) grpc.UnaryClientInterceptor {
	tracer := otel.Tracer(serviceName)

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		spanCtx, span := tracer.Start(ctx, fmt.Sprintf("grpc.%s", method),
			trace.WithSpanKind(trace.SpanKindClient),
			trace.WithAttributes(attribute.String("rpc.system", "grpc"),
				attribute.String("rpc.method", method)),
		)
		defer span.End()

		md, ok := metadata.FromOutgoingContext(spanCtx)
		if !ok {
			md = metadata.MD{}
		}
		Propagator.Inject(spanCtx, metadataCarrier(md))
		opts = append(opts, grpc.Header(&md))

		err := invoker(spanCtx, method, req, reply, cc, opts...)
		if err != nil {
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}
}
