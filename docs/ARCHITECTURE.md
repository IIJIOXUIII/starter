# Архитектурный план — стартовый проект «Rosterra» (рабочее имя)

> Микросервисный стартовый проект: Go-бэкенд + Nuxt 4 / микрофронты + FSD + BFF с генерацией типов.
> Версия плана: 1.1 · Дата: 2026-11-22 · v1.1: библиотека компонентов (Storybook, скриншотные/props/a11y-тесты), e2e критического пути, политика JSDoc
> Точные версии библиотек фиксируются в момент скаффолдинга (M0) и далее обновляются автоматически через Renovate.

---

## 1. Цели и принципы

**Что строим:** абстрактный SaaS-сервис, который начнётся с регистрации/авторизации и админки пользователей, но спроектирован так, чтобы разрастаться до десятков микросервисов и множества микрофронтендов без переписывания фундамента.

**Принципы:**

1. **Contract-first.** Единственный источник правды — OpenAPI 3.1 и protobuf-контракты. Код Go и TS генерируется из них. Фронт и бэк разрабатываются параллельно от одного контракта, расхождение типов невозможно по построению.
2. **Database per service.** Каждый микросервис владеет своей схемой/БД. Никаких прямых чтений чужих таблиц.
3. **BFF-паттерн.** Браузер общается только со своим BFF (Nuxt server routes). Сервисы не выставляют REST наружу.
4. **FSD внутри каждого фронтенд-приложения.** Слои `app → screens → widgets → features → entities → shared`, кросс-импорты запрещены, контроль линтером (steiger).
5. **Observability с первого дня.** Traces, metrics, structured logs — не «потом», а в скелете.
6. **Локальная разработка = прод-топология.** Docker Compose повторяет форму K8s-деплоя; Helm-чарты пишем параллельно и проверяем в kind.
7. **Монорепо.** TS и Go живут в одном репозитории: атомарные изменения контракта + кода в одном PR.
8. **Everything as code:** инфраструктура (Helm), CI/CD, дашборды, алерты, ADR — всё в git.

---

## 2. Высокоуровневая архитектура

```
                                   ┌────────────────────────────────────┐
                                   │              Browser               │
                                   └───────┬────────────────▲───────────┘
                                           │ /              │ JS (MF remotes,
                                           ▼                │ ES-модули)
                              ┌──────────────────────┐      │
                              │  Traefik (edge: TLS, │      │
                              │  routing, headers)   │      │
                              └───┬──────────────┬───┘      │
                                  │ /*           │ /_mfe/*  │
                    ┌─────────────▼──────────┐   │   ┌──────┴───────────────┐
                    │  Shell — Nuxt 4 (SSR)  │   │   │ MFE remotes (Vite):  │
                    │  · layout, роутинг     │   │   │ · mfe-auth (вход/    │
                    │  · auth-guard          │   │   │    регистрация)      │
                    │  · BFF: server/api/**  │   │   │ · mfe-admin (админка)│
                    └───────────┬────────────┘   │   └──────────────────────┘
                                │ /api/** (REST+JSON)
                                │ типобезопасные клиенты из OpenAPI
              ┌─────────────────┴─────────────────┐
              │        Go-микросервисы            │
              │  ┌─────────────┐   ┌────────────┐ │   gRPC (buf) — межсервисные
              │  │ auth-service│◄──┤users-service│ │◄──синхронные вызовы
              │  │ JWT, роли,  │   │ профили,   │ │   NATS JetStream — события
              │  │ сессии      │   │ админ-данные│ │   (user.created …)
              │  └──┬───┬──────┘   └──┬─────────┘ │
              └─────┼───┼─────────────┼───────────┘
                    │   │             │
             PostgreSQL  Redis     PostgreSQL
             (schema:   (сессии,    (schema:
              auth)      refresh)    users)
```

**Ключевые потоки:**

- **Браузер → Shell:** SSR/гидрация, shell подгружает микрофронты через Module Federation 2.0 (remotes — Vite-билды на static hosting).
- **Браузер → BFF:** все API-вызовы идут на `server/api/**` того же origin (нет CORS, cookies httpOnly «из коробки»).
- **BFF → сервисы:** REST/JSON по OpenAPI-контрактам (типобезопасно), JWT в `Authorization`.
- **Сервис → сервис:** синхронно — gRPC; асинхронно — события в NATS JetStream (CloudEvents-конверт).
- **Сервисы сами валидируют JWT** (shared-пакет + JWKS от auth-service) — defense in depth, BFF не является «доверенной зоной».

### Почему нет отдельного Go API Gateway на старте

Роль шлюза для браузера выполняет **BFF** (агрегация, сессии, куки) + **Traefik** (TLS, роутинг, rate limit, security headers). Второй хоп «BFF → Gateway → сервис» дал бы только лишнюю задержку и дублирование. Выделенный Go-gateway появится в фазе эволюции — когда придут **внешние клиенты**: мобильное приложение, публичный partner API (см. §14). Это зафиксируем в ADR-002.

---

## 3. Стек технологий

### 3.1 Фронтенд

| Задача                  | Выбор                                                                               | Почему                                                                       |
| ----------------------- | ----------------------------------------------------------------------------------- | ---------------------------------------------------------------------------- |
| Shell / SSR / BFF       | **Nuxt 4** (Nitro, Vite)                                                            | SSR + серверные роуты в одном приложении, лучший DX во Vue-экосистеме        |
| Микрофронты (remotes)   | **Vue 3.5 + Vite 7 + @module-federation/vite** (MF 2.0)                             | независимо деплоируемые remotes, shared-зависимости синглтонами              |
| FSD-структура и линт    | **Feature-Sliced Design v2 + steiger**                                              | стандарт де-факто для больших Vue/React проектов                             |
| Состояние               | **Pinia 3**                                                                         | официальный стор Vue                                                         |
| Серверные данные        | **@tanstack/vue-query v5**                                                          | кеш, рефетч, invalidation — не тянем серверное состояние в Pinia             |
| Формы + валидация       | **vee-validate 4 + zod**                                                            | типобезопасные схемы, одинаковая валидация на клиенте и в BFF                |
| UI-кит                  | **Tailwind CSS 4 + shadcn-vue (reka-ui) + lucide icons**                            | токены и компоненты в `packages/ui`, единый стиль всех MFE                   |
| Каталог компонентов     | **Storybook 9** (@storybook/vue3-vite, autodocs, addon-a11y, test-runner)           | живая документация дизайн-системы + основа компонентных и скриншотных тестов |
| Версионирование пакетов | **Changesets**                                                                      | семантические версии извлекаемых пакетов (прежде всего `packages/ui`)        |
| Таблицы (админка)       | **@tanstack/vue-table 8**                                                           | пагинация/сортировка/фильтры из коробки                                      |
| Типы API                | **openapi-typescript 7 + openapi-fetch**                                            | типы и клиент генерируются из OpenAPI — ноль ручных интерфейсов              |
| Тесты                   | **Vitest 3 + @vue/test-utils; Playwright (e2e)**                                    | быстрый unit + реальный браузер для критичных сценариев                      |
| Качество                | **ESLint 9 (flat config) + @nuxt/eslint, Prettier, husky, lint-staged, commitlint** | конвенции коммитов и автофиксы                                               |

### 3.2 Бэкенд (Go)

| Задача              | Выбор                                            | Почему                                                                                                   |
| ------------------- | ------------------------------------------------ | -------------------------------------------------------------------------------------------------------- |
| Язык                | **Go 1.25+**                                     | актуальная стабильная ветка на момент скаффолдинга                                                       |
| HTTP-роутер         | **chi v5 + oapi-codegen v2 (strict-server)**     | контракт-first: сервер реализует сгенерированный интерфейс, несовпадение контракта = ошибка компиляции   |
| Межсервисный RPC    | **gRPC + buf** (buf generate / lint / breaking)  | типобезопасные контракты, контроль ломающих изменений в CI                                               |
| PostgreSQL          | **pgx v5 + sqlc**                                | типобезопасный SQL из .sql-файлов, без «магии» ORM                                                       |
| Миграции            | **goose** (embedded)                             | версионированные миграции прямо в бинарнике сервиса                                                      |
| Redis               | **go-redis v9**                                  | refresh-сессии, кеш, rate limit                                                                          |
| Брокер              | **nats.go + JetStream**                          | события между сервисами; репозиторий абстрагирует брокера → Kafka встанет позже без переписывания домена |
| JWT                 | **golang-jwt/jwt v5, EdDSA (Ed25519) + JWKS**    | ключи с `kid` и ротацией; публичные ключи отдаются JWKS-эндпоинтом                                       |
| Хеширование паролей | **alexedwards/argon2id**                         | OWASP-рекомендация, параметры в конфиге                                                                  |
| DI                  | **uber-go/fx**                                   | единый каркас запуска/остановки всех сервисов, модульность                                               |
| Конфиг              | **caarlos0/env + .env**                          | декларативный, 12-factor                                                                                 |
| Логи                | **slog (JSON, stdlib)**                          | structured logs → Loki без адаптеров                                                                     |
| Traces/metrics      | **OpenTelemetry SDK + prometheus/client_golang** | OTLP → Tempo, scrape → Prometheus                                                                        |
| Валидация запросов  | **go-playground/validator** (транспортный слой)  | плюс доменные инварианты в service-слое                                                                  |
| Тесты               | **testify + testcontainers-go**                  | интеграционные тесты с реальными PG/Redis/NATS в Docker                                                  |
| Линт                | **golangci-lint v2**                             | единый конфиг на все сервисы                                                                             |

### 3.3 Инфраструктура

| Задача                            | Выбор                                                                                                                                                       |
| --------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------------------------- |
| Контейнеры / локальная разработка | Docker + **Compose v2** (compose watch — hot reload)                                                                                                        |
| Образы                            | multi-stage, **distroless/static**, non-root, кеш-мOUNTы для Go/pnpm                                                                                        |
| Edge-прокси                       | **Traefik v3** (в K8s — ingress)                                                                                                                            |
| БД                                | **PostgreSQL 18** (схемы `auth`, `users` — физически одна инстанс в dev, разные схемы = database-per-service)                                               |
| Кеш/сессии                        | **Redis 8**                                                                                                                                                 |
| Брокер                            | **NATS 2.11 JetStream**                                                                                                                                     |
| Наблюдаемость                     | OpenTelemetry → **Grafana + Prometheus + Loki + Tempo**, сборщик **Grafana Alloy**                                                                          |
| Kubernetes                        | **Helm 3** (chart-шаблон + umbrella), локальная проверка в **kind**; kustomize — опционально поверх                                                         |
| CI/CD                             | **GitHub Actions** + GHCR; **Trivy** (скан образов), **syft** (SBOM), **oasdiff** (breaking-changes контрактов), **Renovate** (автообновление зависимостей) |
| DX-команды                        | **justfile** (`just dev`, `just test`, `just gen`)                                                                                                          |

---

## 4. Структура монорепозитория

```
E:\newProject\
├── apps/
│   ├── shell/                     # Nuxt 4: SSR-хост MFE + BFF (server/api/**)
│   ├── mfe-auth/                  # Vue3+Vite MF remote: вход, регистрация
│   └── mfe-admin/                 # Vue3+Vite MF remote: админка пользователей
├── packages/
│   ├── ui/                        # дизайн-система: токены, Tailwind-пресет, shadcn-vue компоненты
│   ├── api-client/                # СГЕНЕРИРОВАНО из OpenAPI: типы + openapi-fetch клиенты
│   ├── config-eslint/             # общие flat-config'и
│   ├── config-ts/                 # базовые tsconfig'и
│   └── utils/                     # zod-схемы форм, общие утилиты TS
├── services/
│   ├── auth/                      # Go: регистрация, JWT, сессии, роли
│   └── users/                     # Go: профили, админские данные пользователей
├── pkg/                           # общие Go-библиотеки:
│   │                              #   jwtauth (JWKS-валидация), postgres, redis,
│   │                              #   natsx (брокер-абстракция), otelx, httpx, config, logger
├── api/
│   ├── openapi/                   # auth.yaml, users.yaml — источник правды REST
│   └── proto/                     # buf.yaml, *.proto — межсервисный gRPC
├── deploy/
│   ├── compose/                   # docker-compose.{dev,prod}.yml, .env.example
│   ├── docker/                    # Dockerfile.go-service, Dockerfile.nuxt, Dockerfile.vite-mfe
│   └── helm/                      # service-chart (шаблон), umbrella-chart
├── docs/
│   ├── ARCHITECTURE.md            # этот документ
│   ├── adr/                       # ADR-001…NNN — журнал архитектурных решений
│   └── runbook.md                 # как поднять, отладить, типовые проблемы
├── .github/workflows/             # ci.yml, cd.yml
├── turbo.json                     # пайплайны задач TS
├── pnpm-workspace.yaml
├── go.work                        # Go workspace на все сервисы и pkg/
├── justfile
└── README.md
```

**Оркестрация задач:** pnpm workspaces + Turborepo для TS (кеш, топологический порядок `generate → build → test`), `go.work` для Go (одна IDE-сессия на все сервисы). Go-сборки не идут через Turborepo — их связывает justfile и CI.

---

## 5. Микросервисы (бэкенд)

### 5.1 Состав сервисов на старте

| Сервис            | Ответственность                                                                    | Хранилище                 |
| ----------------- | ---------------------------------------------------------------------------------- | ------------------------- |
| **auth-service**  | регистрация, аутентификация, выпуск/ротация JWT, refresh-сессии, роли (RBAC), JWKS | PostgreSQL `auth` + Redis |
| **users-service** | профили пользователей, статус (active/blocked), админские списки/фильтры           | PostgreSQL `users`        |

Следующие кандидаты (фаза роста): `notifier` (email/telegram), `projects` (домен будущего сервиса), `audit`, `files`.

### 5.2 Шаблон сервиса (чистая архитектура)

```
services/auth/
├── cmd/auth/main.go            # сборка: fx-модули, graceful shutdown
├── internal/
│   ├── config/                 # env-конфиг, валидация при старте
│   ├── domain/                 # сущности (User, Role, Session), доменные ошибки,
│   │                           # интерфейсы репозиториев — БЕЗ зависимостей от инфраструктуры
│   ├── service/                # бизнес-логика: Register, Login, Refresh, Logout, Roles
│   ├── repo/
│   │   ├── postgres/           # sqlc-сгенерированный код + адаптеры
│   │   └── redis/              # refresh-сессии
│   ├── transport/
│   │   ├── http/               # oapi-codegen strict-server (chi): маппинг доменных ошибок → HTTP
│   │   ├── grpc/               # gRPC-сервер для межсервисных вызовов
│   │   └── mq/                 # publishers/consumers NATS
│   └── worker/                 # фоновые задачи (outbox-релей, TTL-чистка)
├── migrations/                 # goose *.sql (embed)
├── sqlc.yaml
└── *_test.go                   # unit + интеграционные (testcontainers-go)
```

Правила зависимостей: `transport → service → domain`, `repo → domain`. Общий код — только через `pkg/`. Никакого импорта `internal/` чужого сервиса.

### 5.3 Контракты и пайплайн генерации типов

```
api/openapi/auth.yaml ──► oapi-codegen ──► Go: типы + strict-server интерфейс (services/auth)
        │
        └─────────────────► openapi-typescript ──► packages/api-client (TS-типы)
                                  │
                                  └──► openapi-fetch клиенты ──► BFF (server/api) типобезопасно

api/proto/*.proto ──► buf generate ──► Go gRPC-стабы (межсервисные вызовы)
```

- Каждый сервис в CI проверяет свой контракт: линт (spectral/buf), **oasdiff** — ломающие изменения блокируют merge.
- `packages/api-client` — только сгенерированный код + тонкая фабрика клиентов (base URL, fetch-обёртка с trace-id). Ручные правки запрещены.

### 5.4 События (NATS JetStream)

- Формат: CloudEvents 1.0 (JSON), subjects: `svc.auth.user.created`, `svc.users.profile.updated`, …
- Паттерн **transactional outbox** (таблица `outbox` + релей в `worker/`) — событие не потеряется при коммите транзакции.
- Каждый потребитель — durable consumer со своим именем; идемпотентность по `event.id`.

Стартовые события: `user.created`, `user.blocked`, `user.unblocked`, `user.role_changed`, `profile.updated`.

---

## 6. BFF и сессии (Nuxt server routes)

`apps/shell/server/api/**` — единственная точка входа API для браузера:

- `server/api/auth/[register|login|refresh|logout|me].post.ts`
- `server/api/users/[list|get|patch].ts` и `server/api/auth/roles/[...].ts` (агрегация auth+users)
- Клиенты — сгенерированные `openapi-fetch` из `packages/api-client`; валидация входных данных — zod-схемы из `packages/utils`.
- **Единый error-envelope** для браузера: `{ code, message, details? }`; доменные ошибки сервисов маппятся один раз в BFF.
- При SSR shell сам читает access-cookie и ходит в сервисы — пользователь не ждёт «водопада» запросов.

---

## 7. Микрофронты + FSD

### 7.1 Состав

| Приложение       | Тип                | Маршруты shell                     | Exposes (MF)                            |
| ---------------- | ------------------ | ---------------------------------- | --------------------------------------- |
| `apps/shell`     | Nuxt 4 (host, SSR) | `/`, layout, меню, guards          | —                                       |
| `apps/mfe-auth`  | Vue3+Vite remote   | `/auth/login`, `/auth/register`    | `AuthPage`, `LoginPage`, `RegisterPage` |
| `apps/mfe-admin` | Vue3+Vite remote   | `/admin/users`, `/admin/users/:id` | `UsersListPage`, `UserEditPage`         |

### 7.2 Механика

- **Module Federation 2.0** (`@module-federation/vite`): remotes — отдельные билды, деплоятся независимо (у каждого свой образ/CDN-путь), shell грузит их в рантайме по манифесту.
- Shared-синглтоны: `vue`, `pinia` — один экземпляр на страницу.
- Каждый remote умеет **standalone-режим** (свой `index.html` + dev-сервер) — разработка MFE независимо от shell.
- Remotes регистрируют свои маршруты в роутере shell через контракт `installRoutes(router)` — навигация сквозная, URL принадлежит shell.
- Ограничение (осознанное, зафиксируем в ADR-003): содержимое remote рендерится **CSR**; SSR остаётся у shell (каркас, мета, данные). Это стандартный компромисс MF.

### 7.3 FSD в каждом MFE и в packages

```
src/
├── app/          # инициализация: провайдеры, роутер, mf-экспорт
├── screens/      # страницы (LoginPage, RegisterPage, UsersListPage…)
├── widgets/      # составные блоки (UserFormCard, UsersTable)
├── features/     # действия пользователя (auth-by-password, update-user-roles)
├── entities/     # доменные сущности фронта (user, session)
└── shared/       # ui (реэкспорт packages/ui), api-клиенты, lib
```

- Кросс-импорты слоёв сверху вниз запрещены; каждый слой — `public API` через `index.ts`.
- **steiger** (FSD-линтер) — обязательный шаг CI.
- Общий UI/токены/клиенты — в `packages/*` (это «shared» уровня всей платформы), MFE импортируют их в своём слое `shared`.

### 7.4 Библиотека компонентов `packages/ui` — с прицелом на вынос в отдельный репозиторий

Библиотека с первого дня проектируется как **извлекаемый пакет**: в любой момент переезжает в свой репозиторий без переписывания и без изменения импортов у потребителей (ADR-007).

- Собственное имя `@rosterra/ui`, собственный `package.json` с `exports`, версии через **Changesets** (semver + changelog).
- **Нулевая связанность с приложениями**: импорты только из внешних зависимостей (vue, reka-ui, tailwind-токены); ни одного относительного импорта в apps/remotes — CI проверяет изоляцию пакета.
- Состав старта: дизайн-токены (CSS-переменные + Tailwind-пресет), примитивы (Button, Input, Label, Checkbox, Select, Dialog, Toast, Table), составные блоки (FormField, DataTable, StatusBadge, UserCard), иконки (lucide реэкспорт).
- Структура компонента: `Button/{Button.vue, Button.stories.ts, Button.test.ts, index.ts}`, public API через корневой `index.ts`.

**Storybook — источник правды и площадка тестов:**

| Тест                                | Инструмент                                                      | Что проверяет                                                                          |
| ----------------------------------- | --------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| Документация                        | CSF3 stories + autodocs                                         | все состояния и пропсы задокументированы                                               |
| **Props/unit**                      | Vitest + @vue/test-utils                                        | рендер по комбинациям пропсов, emits, слоты, граничные значения                        |
| **Interaction**                     | play-функции + `@storybook/test`                                | реальное поведение: ввод, клики, валидация, диалоги                                    |
| **Скриншотные (visual regression)** | Playwright test-runner + `toHaveScreenshot` по сборке Storybook | снапшоты всех stories в git; расхождение = обязательное ревью (опция роста: Chromatic) |
| **A11y**                            | addon-a11y + axe в test-runner                                  | доступность каждого компонента — гейт CI                                               |

---

## 8. Аутентификация и авторизация (JWT)

### 8.1 Модель токенов

| Токен       | Формат                      | TTL     | Хранение у клиента                       | Передача                               |
| ----------- | --------------------------- | ------- | ---------------------------------------- | -------------------------------------- |
| **Access**  | JWT, EdDSA (Ed25519), `kid` | 15 мин  | httpOnly Secure cookie, `Path=/`         | BFF → сервисы: `Authorization: Bearer` |
| **Refresh** | opaque random 32 байта      | 30 дней | httpOnly Secure cookie, `Path=/api/auth` | только BFF → auth-service              |

Claims access-токена: `sub` (user id), `sid` (session id), `roles` (RBAC), `email`, `jti`, `iat`, `exp`, `iss`, `aud`.

**Почему так:**

- Токены **не в localStorage** → XSS не угрожает кражей сессии; куки httpOnly + `SameSite=Lax` + проверка `Origin` на мутациях BFF = защита от CSRF.
- Refresh — **не JWT**, а opaque-строка: в БД хранится только SHA-256 хеш; утечка Redis не выдаёт действительные токены.
- **Ротация refresh** при каждом обновлении + **reuse detection**: повторное использование старого refresh → отзыв всей цепочки сессий (признак кражи).
- **EdDSA + JWKS**: auth-service публикует `/.well-known/jwks.json`; приватный ключ только у auth; остальные сервисы проверяют подпись по публичным ключам с кешем и поддержкой `kid`-ротации. Компрометация одного сервиса не даёт возможность подделывать токены.

### 8.2 Потоки

```
РЕГИСТРАЦИЯ
Браузер ── POST /api/auth/register {email, password, name} ──► BFF (zod)
   BFF ── POST /auth/register ──► auth-service
        Argon2id (пароль) → INSERT users (status=active, email_verified=false)
        → gRPC users.CreateProfile (sync, best-effort + retry)
        → outbox → NATS user.created
   auth ──► access + refresh (создать сессию в Redis)
   BFF ◄── Set-Cookie ×2 ──► браузер (авто-логин, redirect на /)

ОБНОВЛЕНИЕ ДОСТУПА (прозрачно для UI)
Браузер ── любой /api/* ──► BFF: access истёк?
   BFF ── POST /auth/refresh (refresh-cookie) ──► auth-service: ротация + reuse-detection
   BFF ◄── новый access ──► новый Set-Cookie → повтор исходного запроса
```

### 8.3 RBAC

- Роли: `user` (по умолчанию), `admin`. Владелец ролей — **auth-service** (таблицы `roles`, `user_roles`), изменения публикуют `user.role_changed`.
- Проверка тройная: BFF-гвард для UX (редирект), middleware в сервисе (истина в последней инстанции), и на уровне страниц admin-MFE.

### 8.4 Security-чеклист старта

- Argon2id: параметры OWASP (m=64MB, t=3, p=4) в конфиге.
- Rate limit на login/register (per-IP + per-email) на BFF + Traefik.
- Аккаунт-лок после N неудач (cool-down, без раскрытия «пользователь существует»).
- Единые сообщения ошибок логина (не раскрываем существование email).
- Секреты: только через env / `.env` (git-ignored) → в K8s Secret; на проде — SOPS/External Secrets (фаза роста).
- В логах нет PII и токенов; request-id/trace-id в каждой записи.
- Security headers на Traefik: HSTS, X-Content-Type-Options, Referrer-Policy, CSP (report-only на старте).

---

## 9. Страницы (M4–M5)

### Регистрация (`/auth/register`, mfe-auth)

- Поля: email, имя, пароль + подтверждение; zod-схема (длина, сложность), индикатор надёжности пароля.
- Ошибки API маппятся к полям (email занят и т.п.); success → авто-логин → `/`.

### Вход (`/auth/login`, mfe-auth)

- email + пароль, «запомнить меня» (TTL refresh 30 дней vs 12 часов), единая ошибка «Неверный email или пароль», состояние «заблокирован».

### Админка (`/admin/users`, mfe-admin, guard: role=admin)

- Таблица (@tanstack/vue-table): пагинация, поиск по email/имени, фильтр по статусу/роли; колонки: пользователь, статус, роли, создан.
- **Страница редактирования пользователя** (`/admin/users/:id`, отдельный экран):
  - профиль: имя, аватар-URL (файлы — фаза роста), заметки;
  - статус: active / blocked (с причиной) — идёт в auth-service;
  - роли: чек-боксы (user/admin) — идёт в auth-service;
  - история изменений — фаза роста (audit-сервис).

---

## 10. Инфраструктура

### 10.1 Docker Compose (dev)

```
compose (dev):
  traefik          # edge: shell на :80, mfe-манифесты, /api
  shell            # Nuxt (nitro dev, compose watch sync)
  mfe-auth         # vite dev server (standalone MF remote)
  mfe-admin        # vite dev server
  auth-service     # go (air-подобный hot reload)
  users-service    # go
  postgres         # схемы auth/users, init-скрипты
  redis
  nats             # jetstream enabled
  grafana / prometheus / loki / tempo / alloy   # наблюдаемость
```

- Hot reload: `compose watch` (bind sync для TS, restart для Go).
- Один `.env.example` на всё; секреты дев-окружения — не секреты, но структура как на проде.

### 10.2 Образы

- Go: multi-stage (`golang:1.25` → **distroless/static**), non-root, healthcheck `/healthz`.
- Nuxt: node:build-stage → nitro-билд → slim runtime.
- MFE: статика на nginx (unprivileged) / CDN.

### 10.3 Kubernetes (параллельно с dev, проверка в kind)

- `deploy/helm/service-chart` — параметризованный чарт (deployment, service, ingress, HPA, PodDisruptionBudget, healthchecks, resources, env-from-secret).
- `deploy/helm/umbrella` — зависимости: postgres (CloudNativePG-оператор — фаза роста; на старте stateful), redis, nats, traefik + все наши сервисы.
- Проверка: `kind create cluster && helm install` в CI (nightly).
- GitOps (ArgoCD/Flux) — фаза роста, когда появится прод-кластер.

### 10.4 Окружения

| Среда | Подъём                            | Данные            |
| ----- | --------------------------------- | ----------------- |
| dev   | Compose на машине разработчика    | seed-скрипты      |
| stage | kind или сервер с k3s + Helm      | анонимизированные |
| prod  | managed K8s + Helm (позже ArgoCD) | боевые            |

---

## 11. Наблюдаемость

- **Traces:** OTel SDK во всех Go-сервисах (`otelhttp`/`otelgrpc`) + BFF (nitro-хуки) → Tempo; сквозной trace-id от браузера до SQL.
- **Metrics:** Prometheus (RED-метрики HTTP/gRPC, бизнес-метрики: registrations_total, logins_total) + дашборды Grafana as code.
- **Logs:** slog JSON → Loki через Alloy; корреляция по trace_id.
- Health: `/healthz` (liveness) и `/readyz` (readiness с проверкой PG/Redis/NATS) — используются и Compose, и K8s.
- Алерты: сначала базовые (5xx-rate, latency p99, restart loops) — правила в git.

---

## 12. CI/CD и качество

**GitHub Actions:**

- `ci.yml` — по PR, с детекцией изменённых путей:
  - TS: `turbo lint test build` + steiger + tsc --noEmit; Storybook: сборка + test-runner (interaction/a11y) + скриншотные снапшоты (расхождение = fail с diff-артефактом);
  - Go: `golangci-lint run`, `go test -race ./...` (testcontainers поднимает PG/Redis/NATS);
  - контракты: spectral/buf lint + **oasdiff** (breaking → fail);
  - docker build (кеш buildx) + **Trivy** (fail на HIGH/CRITICAL) + **syft** (SBOM);
  - `helm lint` + деплой smoke в kind (nightly).
- `cd.yml` — push в `main`: образы → GHCR (теги semver+sha), деплой stage.

**Локальные гейты:** husky + lint-staged (eslint/prettier на изменённых файлах, golangci-lint --fix), commitlint (Conventional Commits) — история пригодна для автогенерации changelog и semver.

**Тест-пирамида:**

| Уровень                    | Инструмент                                 | Покрывает                                                                                            | Обязательность         |
| -------------------------- | ------------------------------------------ | ---------------------------------------------------------------------------------------------------- | ---------------------- |
| unit (Go)                  | testify                                    | домен, сервисы, `pkg/`                                                                               | гейт CI                |
| интеграция (Go)            | testcontainers-go                          | репозитории против реальных PG/Redis/NATS, HTTP по контракту                                         | гейт CI                |
| unit (TS)                  | Vitest + @vue/test-utils                   | **shared-функции (JSDoc + `@example`)**, entities/features FSD, **пропсы/события/слоты компонентов** | гейт CI                |
| interaction-тесты          | Storybook play-функции (@storybook/test)   | поведение компонентов в реальном браузере                                                            | гейт CI                |
| **скриншотные**            | Playwright `toHaveScreenshot` по Storybook | визуальная регрессия всех состояний                                                                  | гейт CI (diff → ревью) |
| a11y                       | addon-a11y + axe                           | доступность компонентов                                                                              | гейт CI                |
| **e2e — критический путь** | Playwright                                 | **регистрация → вход → редактирование пользователя в админке**                                       | гейт CI на main        |
| нагрузка                   | k6                                         | smoke auth-эндпоинтов                                                                                | по требованию          |

**Политика документации кода:**

- TS: все экспортируемые функции `packages/*` и слоя `shared` — с JSDoc (`@param`, `@returns`, `@example`); контроль — `eslint-plugin-jsdoc` в CI.
- Go: godoc-комментарии на всех экспортированных символах `pkg/`; контроль — revive в golangci-lint.

---

## 13. Дорожная карта (этапы реализации)

| Этап                      | Содержание                                                                                                                                                                                        | Результат                                           |
| ------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | --------------------------------------------------- |
| **M0. Скелет**            | git init; pnpm+turbo+tsconfig+eslint+prettier+husky+commitlint; go.work+golangci-lint; justfile; compose c PG/Redis/NATS; README; ADR-001…003                                                     | «пусто, но всё линтится, тестируется и поднимается» |
| **M1. Контракты**         | api/openapi/auth.yaml, users.yaml; buf + proto; пайплайн oapi-codegen / openapi-typescript; packages/api-client                                                                                   | типы Go и TS из одного источника                    |
| **M2. auth-service**      | миграции; Argon2id; register/login/refresh/logout/me; JWKS; outbox→NATS; тесты testcontainers; Dockerfile                                                                                         | рабочий бэкенд аутентификации                       |
| **M3. BFF**               | server/api (сессии-cookie, ротация refresh, error-envelope); клиенты api-client; rate limit                                                                                                       | браузерная сессия работает end-to-end               |
| **M4. Shell + mfe-auth**  | MF-инфраструктура (host/remotes, standalone); **Storybook + базовые компоненты `packages/ui` с полным набором тестов (props/unit, interaction, скриншотные, a11y)**; формы login/register; guards | страницы живые + каталог компонентов с тестами      |
| **M5. users + mfe-admin** | users-service (профили, списки, статусы); admin-MFE: таблица + страница редактирования; RBAC-гварды                                                                                               | админка готова                                      |
| **M6. Наблюдаемость**     | OTel во всех сервисах+BFF; Prometheus/Loki/Tempo/Grafana в compose; дашборды; health/ready                                                                                                        | видимость сквозь весь стек                          |
| **M7. Helm/K8s**          | service-chart + umbrella; проверка в kind; ingress; secrets                                                                                                                                       | поднимается и в Compose, и в K8s                    |
| **M8. Полировка**         | Playwright e2e критического пути (регистрация→вход→админка); visual-regression гейт в CI; k6 smoke; JSDoc-покрытие shared; ADR-004…007; runbook; Renovate; SBOM                                   | «идеальный стартовый проект» — готов                |

Порядок осознанный: контракты (M1) раньше кода — фронт и бэк дальше идут параллельно; K8s (M7) после того, как образы и health-checks стабилизировались.

---

## 14. Фаза роста (заложено, не строим сейчас)

- Email-верификация, восстановление пароля, OAuth (Google/GitHub), 2FA (TOTP).
- `notifier` (email/telegram) — первый новый микросервис по шаблону M2, proof of scalability.
- Выделенный **Go API Gateway** — при появлении мобильных клиентов/партнёрского API (см. §2).
- Kafka/Redpanda **за существующей брокерной абстракцией** `pkg/natsx` (переписывания домена не потребуется).
- ClickHouse (аналитика), audit-сервис, feature flags (OpenFeature), i18n (@nuxtjs/i18n).
- CloudNativePG-оператор, ArgoCD GitOps, service mesh (linkerd) при >10 сервисах.
- Мультитенантность, биллинг-MFE — по требованиям «абстрактного сервиса».

---

## 15. Риски и осознанные компромиссы

| Риск                                            | Митигация                                                                                                        |
| ----------------------------------------------- | ---------------------------------------------------------------------------------------------------------------- |
| Зрелость MF 2.0 + Vite для Vue                  | проверяем в M4 на минималках; fallback-план — single-spa / native import maps (ADR-003)                          |
| SSR только у shell, remotes — CSR               | компромисс задокументирован; критичные данные тянутся SSR-слоем shell                                            |
| Nuxt-структура папок vs FSD в shell             | FSD строго внутри MFE и packages; pages shell — тонкие композиции                                                |
| Windows-разработка                              | Docker Desktop/WSL2; `compose watch`; все скрипты — кроссплатформенный justfile                                  |
| Схемы в одном PG-инстансе ≠ физическая изоляция | для dev достаточно; на проде — отдельные БД/инстансы (CloudNativePG), доступа между сервисами нет по credentials |

---

## 16. ADR (записать в M0–M1)

1. ADR-001: Монорепо pnpm+Turborepo+go.work (альтернатива: Nx, полиглот-репо).
2. ADR-002: BFF вместо отдельного API Gateway на старте.
3. ADR-003: Module Federation 2.0 для микрофронтов, CSR-remotes.
4. ADR-004: JWT-модель — EdDSA, opaque refresh, httpOnly cookies, JWKS.
5. ADR-005: NATS JetStream + transactional outbox; абстракция брокера.
6. ADR-006: Contract-first OpenAPI 3.1 + oapi-codegen strict / openapi-typescript.
7. ADR-007: `packages/ui` — извлекаемая библиотека компонентов (Changesets, изоляция импортов, Storybook как площадка тестов).
