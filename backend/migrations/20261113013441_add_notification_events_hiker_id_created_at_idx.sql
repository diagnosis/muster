-- +goose Up
-- +goose StatementBegin
CREATE INDEX notification_events_hiker_id_created_at_idx ON notification_events(hiker_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX notification_events_hiker_id_created_at_idx;
-- +goose StatementEnd
