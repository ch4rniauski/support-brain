-- +goose Up
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    login         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    role          TEXT NOT NULL,
    CONSTRAINT users_login_key UNIQUE (login),
    CONSTRAINT users_role_check CHECK (role IN ('client', 'operator'))
);

-- +goose Down
DROP TABLE IF EXISTS users;
