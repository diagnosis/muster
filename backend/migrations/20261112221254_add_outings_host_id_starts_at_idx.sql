-- +goose Up
-- +goose StatementBegin
CREATE INDEX outings_host_id_starts_at_idx ON outings(host_id, starts_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX outings_host_id_starts_at_idx;
-- +goose StatementEnd
