# Runbook — как поднять и отладить проект

## Требования

| Инструмент              | Версия       | Примечание                                                                                    |
| ----------------------- | ------------ | --------------------------------------------------------------------------------------------- |
| Node.js                 | 22+          | на машине проекта установлен 24.x                                                             |
| pnpm                    | 10+          | установлен глобально (`pnpm --version`)                                                       |
| Go                      | 1.25+        | установлен в `%LOCALAPPDATA%\Programs\go`; проверьте `go version` (нужен в PATH пользователя) |
| Docker Desktop / docker | с compose v2 | `docker compose version`                                                                      |
| just (опционально)      | —            | `justfile` дублирует все команды через pnpm scripts                                           |

## Первый запуск

```powershell
pnpm install                 # TS-зависимости (после установки автоматически: husky, nuxt prepare)
go build ./...               # проверка Go-workspace (go.work)
docker compose -f deploy/compose/docker-compose.dev.yml up -d   # PG/Redis/NATS
pnpm dev                     # shell :3000, mfe-auth :5173, mfe-admin :5174
```

Проверки:

- http://localhost:3000 — shell;
- http://localhost:3000/api/health — BFF health-check;
- `curl http://localhost:8081/healthz` — auth-service (запуск: `go run ./services/auth/cmd/auth`);
- `curl http://localhost:8082/healthz` — users-service (`go run ./services/users/cmd/users`);
- NATS monitor: http://localhost:8222

## Генерация кода (M1+)

```powershell
pnpm gen:api    # TS-типы из OpenAPI → packages/api-client
just gen-go     # Go strict-server (oapi-codegen) для auth/users
just gen-proto  # Go gRPC-стабы (buf) — требует buf и protoc-плагинов (go install)
```

## Запуск микрофронтов (M4+)

Терминология: **shell** (:3000) — Nuxt-хост; **remotes** (:5173 mfe-auth, :5174 mfe-admin) —
Vite-приложения, грузимые shell'ом через Module Federation (mf-manifest.json).

```powershell
# Терминал 1-2: remotes (или build + preview из собранного dist)
pnpm --filter @rosterra/mfe-auth dev      # :5173
pnpm --filter @rosterra/mfe-admin dev     # :5174
# Терминал 3: shell
pnpm --filter @rosterra/shell dev         # :3000
```

Проверки федерации: `curl http://localhost:5173/mf-manifest.json`,
страница `http://localhost:3000/auth/login` должна показать форму (остров).

Скриншотные тесты дизайн-системы:

```powershell
pnpm --filter @rosterra/ui test:storybook:visual   # сборка SB + playwright-снапшоты
pnpm --filter @rosterra/ui test:storybook          # interaction + a11y (test-runner)
# Требуется браузер: pnpm --filter @rosterra/ui exec playwright install chromium
```

Федерация end-to-end (браузер; нужны запущенные shell/remotes/auth):

```powershell
cd apps/shell && pnpm exec playwright test -c playwright.m4.config.ts   # M4: острова входа/регистрации
cd apps/shell && pnpm exec playwright test -c playwright.m5.config.ts   # M5: живая админка
```

## Kubernetes / Helm (M7)

Чарт: `deploy/helm/rosterra` (auth, users, shell, remotes-статика, Ingress, Secret,
опциональный ServiceMonitor). Инфраструктура (PG/Redis/NATS/Jaeger/Prometheus)
подключается внешне — через `values.yaml` (endpoint'ы) или существующие чарты.

```powershell
# Валидация без кластера
helm lint deploy/helm/rosterra
helm template rosterra deploy/helm/rosterra          # рендер всех манифестов

# Сборка образов (контекст — корень монорепо)
docker build -f services/auth/Dockerfile -t rosterra/auth:m7 .
docker build -f services/users/Dockerfile -t rosterra/users:m7 .
docker build -f apps/shell/Dockerfile -t rosterra/shell:m7 .
docker build -f apps/mfe-auth/Dockerfile -t rosterra/mfe-auth:m7 .
docker build -f apps/mfe-admin/Dockerfile -t rosterra/mfe-admin:m7 .

# Установка (когда кластер доступен: Docker Desktop K8s / kind / кластер)
helm install rosterra deploy/helm/rosterra -f deploy/helm/rosterra/values-dev.yaml
```

Ключевые маршруты ingress: `/` → shell, `/mfe/auth/*` → remotes-auth, `/mfe/admin/*`
→ remotes-admin (same-origin MF: куки и манифесты без CORS). Секреты (JWT_SEED,
DSN) — Secret из values; в prod переопределять `--set secrets.*` или своим Secret.

Нюанс сборки Go: go.work перечисляет оба сервиса — в Dockerfile копируется
go.mod «чужого» сервиса (без исходников). pnpm-сборки требуют `onlyBuiltDependencies`
из pnpm-workspace.yaml (esbuild/@swc/core) — в контейнере отрабатывают автоматически.

## CI (M8)

Workflow `.github/workflows/ci.yml` — четыре job'а:

1. **ts** — lint / unit / build всего монорепо (turbo)
2. **go** — build / vet / test по модулям + golangci-lint (testcontainers работают на GH-раннерах)
3. **helm** — lint + template чарта (helm v3.16.4)
4. **e2e** — полная эмуляция dev-стека: postgres/redis/nats как service-контейнеры,
   сборка Go + фронта, запуск auth/users/remotes/shell, Playwright-прогон M4+M5+M8
   (Chromium), артефакты при падении.

Локально все наборы — см. разделы выше; e2e идентичны CI-шагам.

## Наблюдаемость (M6)

```powershell
docker compose -f deploy/compose/docker-compose.dev.yml up -d   # включая jaeger/prometheus/grafana
```

| Компонент   | Где                                   | Что смотреть                                      |
| ----------- | ------------------------------------- | ------------------------------------------------- |
| Jaeger (UI) | http://localhost:16686                | трейсы; сервисы `rosterra/auth`, `rosterra/users` |
| Prometheus  | http://localhost:9090                 | таргеты auth/users (`/metrics`), алерты — M7+     |
| Grafana     | http://localhost:3001 (admin/grafana) | дашборды; datasource'ы преднастроены              |

Трассировка включается переменной `OTEL_ENDPOINT=http://localhost:4318` при запуске
Go-сервисов (пусто → noop). Метрики доступны всегда: `:8081/metrics`, `:8082/metrics`.

Сквозной трейс (проверено e2e): `POST /api/auth/register` (браузер → BFF → traceparent)
→ `[rosterra/auth] POST /v1/register` → outbox-событие (CE-extension traceparent)
→ `[rosterra/users] users.event.consume user.created` — один traceID на все три звена.
Бизнес-счётчики: `auth_register_total`, `auth_login_total`, `auth_refresh_total`,
`users_profiles_created_total`, `http_requests_total{route}`.

## users-service (M5)

Профили пользователей: HTTP `:8082` (strict-server users.yaml), gRPC `:9082` (users.v1),
consumer событий auth (`svc.auth.user.created` → создание профиля, dedup по event id),
outbox-релей `svc.users.>` (stream `ROSTERRA_USERS`).

```powershell
$env:Path = "C:\Users\zaits\AppData\Local\Programs\go\bin;$env:Path"
go build -o $env:TEMP\users-m5.exe ./cmd/users   # в services/users
$env:JWT_SEED = '00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff'
$env:POSTGRES_DSN = 'postgres://rosterra:rosterra@localhost:55432/rosterra?sslmode=disable'
$env:NATS_URL = 'nats://localhost:4222'
& "$env:TEMP\users-m5.exe"
```

Нюанс: в dev auth и users живут в одной БД — миграции users нумеруются с 10000
(общий goose_db_version), таблицы users-сервиса: `profiles`, `processed_events`,
`users_outbox` (у auth — своя `outbox`).

> **Важно:** после пересборки shell (`pnpm build`) перезапустите его процесс
> (`node apps/shell/.output/server/index.mjs`) — запущенный nitro держит
> старые имена чанков в памяти.

## Типовые проблемы (Windows)

- **`go` не найден в новой консоли** — перезапустите терминал после добавления
  в PATH или добавьте `%LOCALAPPDATA%\Programs\go\bin` вручную.
- **pnpm заблокировал build-скрипты** — разрешённые пакеты перечислены в
  `pnpm-workspace.yaml` (`onlyBuiltDependencies`); при установке используйте
  `pnpm install --allow-build=esbuild` (или `pnpm approve-builds`).
- **Порты заняты** — переопределите через `deploy/compose/.env` (шаблон —
  `.env.example`).
- **Долгий первый `pnpm install`** — норма (Nuxt + Vite); кеш pnpm store
  ускоряет последующие установки.

## Установленное окружение на этой машине

- Go 1.27.4 — `%LOCALAPPDATA%\Programs\go` (путь добавлен в user PATH).
- pnpm 12.21.0 — глобальный npm-пакет.
- buf 1.75.0, protoc-gen-go, protoc-gen-go-grpc — `%USERPROFILE%\go\bin`.
- golangci-lint — локально не ставился (проверяет CI); при желании:
  `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`.
- Docker Desktop — переустановлен, работает (compose v2).

## Важно: нативный PostgreSQL на 5432

На машине установлен нативный PostgreSQL 10 (служба Windows), занимающий
порт 5432. Поэтому compose публикует контейнерный PG на **55432**
(см. `deploy/compose/.env`). DSN для локального запуска сервисов:
`postgres://rosterra:rosterra@localhost:55432/rosterra?sslmode=disable`.

## Запуск auth-service локально (M2)

```powershell
# переменные — см. services/auth/.env.example (JWT_SEED обязателен для стабильного JWKS)
$env:POSTGRES_DSN = 'postgres://rosterra:rosterra@localhost:55432/rosterra?sslmode=disable'
$env:JWT_SEED = '00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff'
go run ./services/auth/cmd/auth
```

Проверки: `GET :8081/healthz`, `GET :8081/readyz`, `POST :8081/v1/register`,
`GET :8081/.well-known/jwks.json`. События user.created и т.п. идут в
NATS JetStream (стрим ROSTERRA_AUTH).
