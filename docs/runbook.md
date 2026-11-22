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

## Типовые проблемы (Windows)

- **`go` не найден в новой консоли** — перезапустите терминал после добавления
  в PATH или добавьте `%LOCALAPPDATA%\Programs\go\bin` вручную.
- **pnpm заблокировал build-скрипты** — разрешённые пакеты перечислены в
  `pnpm-workspace.yaml` (`onlyBuiltDependencies`); после изменения выполните
  `pnpm rebuild <имя>`.
- **Порты заняты** — переопределите через `deploy/compose/.env` (шаблон —
  `.env.example`).
- **Долгий первый `pnpm install`** — норма (Nuxt + Vite); кеш pnpm store
  ускоряет последующие установки.
