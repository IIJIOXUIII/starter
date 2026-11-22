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
