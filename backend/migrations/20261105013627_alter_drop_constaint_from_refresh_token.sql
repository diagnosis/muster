-- +goose Up
-- +goose StatementBegin
ALTER TABLE user_refresh_tokens DROP CONSTRAINT uq_hiker_platform;
ALTER TABLE user_refresh_tokens
    ADD COLUMN previous_hash TEXT,
    ADD COLUMN rotated_at    TIMESTAMPTZ;
CREATE UNIQUE INDEX uq_refresh_token_hash ON user_refresh_tokens (token_hash);
CREATE INDEX ix_refresh_previous_hash ON user_refresh_tokens (previous_hash) WHERE previous_hash IS NOT NULL;
CREATE INDEX ix_refresh_hiker ON user_refresh_tokens (hiker_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX ix_refresh_hiker;
DROP INDEX ix_refresh_previous_hash;
DROP INDEX uq_refresh_token_hash;
ALTER TABLE user_refresh_tokens DROP COLUMN rotated_at, DROP COLUMN previous_hash;
DELETE FROM user_refresh_tokens;
ALTER TABLE user_refresh_tokens ADD CONSTRAINT uq_hiker_platform UNIQUE (hiker_id, platform);
-- +goose StatementEnd