-- +goose Up
-- +goose StatementBegin
ALTER TABLE join_requests ADD COLUMN requested_at TIMESTAMPTZ NOT NULL DEFAULT now();
UPDATE join_requests SET requested_at = updated_at WHERE status = 'requested';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE join_requests DROP COLUMN requested_at;
-- +goose StatementEnd
