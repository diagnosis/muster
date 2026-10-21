-- +goose Up
-- +goose StatementBegin
ALTER TABLE auth_tokens ADD COLUMN attempts INT NOT NULL DEFAULT 0;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE auth_tokens DROP COLUMN attempts;
-- +goose StatementEnd
