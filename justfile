# DX-команды проекта. `just` опционален — все рецепты дублируются в package.json scripts.
# Windows: just использует powershell (см. windows-shell ниже).

set windows-shell := ["powershell.exe", "-NoProfile", "-Command"]

default:
    @just --list

# Установить зависимости TS-части
install:
    pnpm install

# Поднять всю дев-инфраструктуру и запустить dev-серверы
dev: up
    pnpm dev

# Собрать все TS-пакеты и приложения
build:
    pnpm build

# Линтеры: ESLint (TS/Vue) + golangci-lint (Go)
lint:
    pnpm lint
    go vet ./...

# Все тесты: Vitest + Go
test:
    pnpm test
    go test ./...

# Инфраструктура: PG / Redis / NATS / наблюдаемость
up:
    docker compose -f deploy/compose/docker-compose.dev.yml up -d

down:
    docker compose -f deploy/compose/docker-compose.dev.yml down

logs service="":
    docker compose -f deploy/compose/docker-compose.dev.yml logs -f {{service}}

# Сгенерировать TS-типы из OpenAPI (packages/api-client)
gen-ts:
    pnpm gen:api

# Сгенерировать Go-код из OpenAPI (strict-server) для обоих сервисов
gen-go:
    cd services/auth && go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi/auth.yaml
    cd services/users && go tool oapi-codegen -config oapi-codegen.yaml ../../api/openapi/users.yaml

# Сгенерировать Go gRPC-стабы из proto (buf) — buf ищет buf.gen.yaml в текущем каталоге
gen-proto:
    cd api/proto; buf generate
