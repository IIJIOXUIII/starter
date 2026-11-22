-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id              uuid PRIMARY KEY,
    email           citext       NOT NULL UNIQUE,
    password_hash   text         NOT NULL,
    name            text,
    status          text         NOT NULL DEFAULT 'active'
                    CHECK (status IN ('active', 'blocked')),
    email_verified  boolean      NOT NULL DEFAULT false,
    created_at      timestamptz  NOT NULL DEFAULT now(),
    updated_at      timestamptz  NOT NULL DEFAULT now()
);

CREATE TABLE user_roles (
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role    text NOT NULL CHECK (role IN ('user', 'admin')),
    PRIMARY KEY (user_id, role)
);

CREATE INDEX idx_user_roles_user ON user_roles (user_id);

-- Transactional outbox: событие пишется в одной транзакции с изменением,
-- релей публикует в NATS и помечает published_at.
CREATE TABLE outbox (
    id           uuid        PRIMARY KEY,
    type         text        NOT NULL,
    payload      jsonb       NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now(),
    published_at timestamptz
);

CREATE INDEX idx_outbox_unpublished ON outbox (created_at) WHERE published_at IS NULL;

-- +goose Down
DROP TABLE IF EXISTS outbox;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
