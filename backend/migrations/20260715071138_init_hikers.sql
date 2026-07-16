-- +goose Up
-- +goose StatementBegin
CREATE TABLE hikers (
        id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        email         TEXT UNIQUE NOT NULL,
        password_hash TEXT NOT NULL,
        name          TEXT NOT NULL,
        experience    TEXT NOT NULL CHECK (experience IN ('beginner','intermediate','experienced')),
        pace          TEXT NOT NULL CHECK (pace IN ('relaxed','moderate','fast')),
        home_area     TEXT,
        bio           TEXT,
        gender        TEXT,
        created_at    TIMESTAMPTZ DEFAULT now(),
        updated_at    TIMESTAMPTZ DEFAULT now(),
        deleted_at    TIMESTAMPTZ
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE user_refresh_tokens (
        id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        hiker_id   UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
        token_hash TEXT NOT NULL,
        platform   TEXT NOT NULL DEFAULT 'web',
        expires_at TIMESTAMPTZ NOT NULL,
        created_at TIMESTAMPTZ DEFAULT now(),
        CONSTRAINT uq_hiker_platform UNIQUE (hiker_id, platform)
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_refresh_tokens;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE IF EXISTS hikers;
-- +goose StatementEnd