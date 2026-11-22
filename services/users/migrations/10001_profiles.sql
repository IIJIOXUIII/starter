-- +goose Up
-- Профили пользователей. user_id = id пользователя в auth-service (users.id),
-- ссылочную целостность между сервисами не ставим: базы логически разделены.
-- Номер миграции 10001+: goose_db_version общая с auth-service (та же БД),
-- версии auth начинаются с 1, users — с 10000.

CREATE TABLE profiles (
    user_id      uuid PRIMARY KEY,
    display_name text        NOT NULL DEFAULT '',
    avatar_url   text,
    notes        text,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Идемпотентность consumer'а: какие события auth уже применены (at-least-once → dedup).
CREATE TABLE processed_events (
    event_id     uuid        PRIMARY KEY,
    processed_at timestamptz NOT NULL DEFAULT now()
);

-- Transactional outbox: profile.updated пишется в одной транзакции с изменением,
-- релей публикует в NATS (svc.users.>) и помечает published_at.
-- Таблица названа users_outbox: в dev auth и users живут в одной БД,
-- у auth своя таблица outbox.
CREATE TABLE users_outbox (
    id           uuid        PRIMARY KEY,
    type         text        NOT NULL,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX idx_users_outbox_unpublished ON users_outbox (created_at) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS users_outbox;
DROP TABLE IF EXISTS processed_events;
DROP TABLE IF EXISTS profiles;
