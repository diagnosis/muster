-- +goose Up
-- +goose StatementBegin
CREATE INDEX join_requests_hiker_id_idx ON join_requests (hiker_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX join_requests_hiker_id_idx;
-- +goose StatementEnd
