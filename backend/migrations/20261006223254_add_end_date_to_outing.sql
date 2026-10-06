-- +goose Up
-- +goose StatementBegin
ALTER TABLE outings
    ADD COLUMN ends_at TIMESTAMPTZ
        CONSTRAINT outings_ends_after_start CHECK (ends_at IS NULL OR ends_at > starts_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE outings DROP COLUMN ends_at;
-- +goose StatementEnd
