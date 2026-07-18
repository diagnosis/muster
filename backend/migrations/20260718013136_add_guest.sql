-- +goose Up
-- +goose StatementBegin
ALTER TABLE join_requests ADD COLUMN guests INT NOT NULL DEFAULT 0
    CHECK (guests >= 0 AND guests <= 3);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE join_requests DROP COLUMN guests;
-- +goose StatementEnd
