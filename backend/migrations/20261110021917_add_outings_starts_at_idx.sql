-- +goose Up
-- +goose StatementBegin
CREATE INDEX outings_starts_at_id_idx ON outings (starts_at, id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX outings_starts_at_id_idx;
-- +goose StatementEnd
