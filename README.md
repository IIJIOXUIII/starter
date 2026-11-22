# Rosterra — стартовый проект микросервисной платформы

Монорепозиторий: **Go-микросервисы** + **Nuxt 4 / микрофронты (FSD)** + **BFF с генерацией типов**.

Полный архитектурный план: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) · Решения: [docs/adr/](docs/adr/) · Эксплуатация: [docs/runbook.md](docs/runbook.md)

## Статус реализации

| Этап                                                                                          | Статус |
| --------------------------------------------------------------------------------------------- | ------ |
| M0 скелет (монорепо, CI, compose)                                                             | ✅     |
| M1 контракты + генерация типов (OpenAPI → Go/TS, proto → gRPC)                                | ✅     |
| M2 auth-service (Argon2id, JWT+JWKS, ротация refresh, reuse detection, outbox→NATS)           | ✅     |
| M3 BFF (httpOnly cookie-сессии, прозрачный refresh, rate limit, CSRF guard)                   | ✅     |
| M4 микрофронты (MF 2.0 islands), Storybook (interaction/a11y/visual), формы входа/регистрации | ✅     |
| M5 users-service + админка (таблица, редактирование)                                          | ⏳     |
| M6 наблюдаемость (OTel, Grafana) · M7 Helm/K8s · M8 e2e-полировка                             | ⏳     |

Что уже работает end-to-end: регистрация → вход → cookie-сессия → прозрачный refresh →
выход; JWKS; события user.*/role_changed в NATS; Storybook дизайн-системы с тремя
видами тестов компонентов; федерация островов в реальном браузере.

## Структура

```
apps/
  shell/            Nuxt 4 — SSR-хост микрофронтов + BFF (server/api)
  mfe-auth/         Vue 3 + Vite — микрофронт входа/регистрации
  mfe-admin/        Vue 3 + Vite — микрофронт админки пользователей
packages/
  ui/               дизайн-система (извлекаемый пакет, Storybook — с M4)
  api-client/       СГЕНЕРИРОВАНО из OpenAPI: типы + типобезопасные клиенты
  utils/            общие TS-утилиты (JSDoc обязателен)
  config-ts/        базовые tsconfig
  config-eslint/    общие ESLint flat-config
services/
  auth/             Go — регистрация, JWT, сессии, роли
  users/            Go — профили пользователей
pkg/               общие Go-библиотеки (logger, httpx, configx)
api/
  openapi/          контракты REST — источник правды
  proto/            контракты gRPC (buf)
deploy/
  compose/          docker-compose для локальной разработки
  docker/           Dockerfile'ы (M2+)
  helm/             чарты Kubernetes (M7)
docs/              архитектура, ADR, runbook
```

## Быстрый старт

Требования: Node 22+, pnpm 10+, Go 1.25+, Docker Desktop (или WSL2 + docker), `just` (опционально).

```bash
pnpm install          # зависимости TS
just up               # инфраструктура: PostgreSQL, Redis, NATS (docker compose)
pnpm dev              # dev-серверы: shell :3000, mfe-auth :5173, mfe-admin :5174
go build ./...        # Go-сервисы (через go.work)
just test             # все тесты
```

Инфраструктура (без `just`):

```bash
pnpm compose:up       # поднять PG/Redis/NATS
pnpm compose:down     # остановить
```

## Команды

| Команда                                  | Что делает                                               |
| ---------------------------------------- | -------------------------------------------------------- |
| `pnpm dev`                               | dev-серверы всех фронтенд-приложений (turbo)             |
| `pnpm build` / `pnpm lint` / `pnpm test` | сборка / линтеры / тесты (turbo, с кешем)                |
| `pnpm typecheck`                         | tsc / vue-tsc по всем TS-пакетам                         |
| `pnpm gen:api`                           | сгенерировать TS-типы из OpenAPI → `packages/api-client` |
| `just gen-go`                            | сгенерировать Go strict-server из OpenAPI (oapi-codegen) |
| `just gen-proto`                         | сгенерировать Go gRPC-стабы (buf)                        |

## Контракты — источник правды

```
api/openapi/*.yaml ──► oapi-codegen ──► Go strict-server (типы + интерфейс)
                  └──► openapi-typescript ──► packages/api-client (TS)
api/proto/*.proto  ──► buf generate ──► Go gRPC-стабы
```

Ручные правки сгенерированного кода запрещены. Ломающие изменения контрактов блокируются в CI (oasdiff — с M2+).

## Коммиты

Conventional Commits (`feat:`, `fix:`, `chore:` …) — контролируется commitlint'ом.
