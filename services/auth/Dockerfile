# auth-service: multi-stage сборка из корня монорепо (нужен go.work: сервис
# зависит от workspace-модуля github.com/rosterra/platform/pkg).
# Сборка: docker build -f services/auth/Dockerfile -t rosterra/auth .
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY pkg/ pkg/
COPY services/auth/ services/auth/
# go.work перечисляет оба сервиса: второй нужен только как go.mod (workspace-вход).
COPY services/users/go.mod services/users/go.mod
WORKDIR /src/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/auth ./cmd/auth

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/auth /usr/local/bin/auth
USER app
EXPOSE 8081 9082
ENTRYPOINT ["/usr/local/bin/auth"]
