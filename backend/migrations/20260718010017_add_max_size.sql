-- +goose Up
-- +goose StatementBegin
-- Re-adds the headcount cap dropped in 0003 — seat capacity and event
-- cap turned out to be two independent constraints. DEFAULT backfills
-- dev rows only; Create always supplies the value.
ALTER TABLE outings ADD COLUMN max_size INT NOT NULL DEFAULT 2 CHECK (max_size >= 2);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE outings DROP COLUMN max_size;
-- +goose StatementEnd