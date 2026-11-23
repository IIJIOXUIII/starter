# users-service: multi-stage сборка из корня монорепо (go.work + pkg).
# Сборка: docker build -f services/users/Dockerfile -t rosterra/users .
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.work go.work.sum ./
COPY pkg/ pkg/
# go.work перечисляет оба сервиса: второй нужен только как go.mod (workspace-вход).
COPY services/auth/go.mod services/auth/go.mod
COPY services/users/ services/users/
WORKDIR /src/services/users
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/users ./cmd/users

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 10001 app
COPY --from=build /out/users /usr/local/bin/users
USER app
EXPOSE 8082 9082
ENTRYPOINT ["/usr/local/bin/users"]
