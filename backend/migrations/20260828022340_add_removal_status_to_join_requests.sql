-- +goose Up
-- +goose StatementBegin
ALTER TABLE join_requests
DROP CONSTRAINT join_requests_status_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE join_requests
    ADD CONSTRAINT join_requests_status_check
        CHECK (status IN ('requested','accepted','declined','withdrawn','removed'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE join_requests
DROP CONSTRAINT join_requests_status_check;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE join_requests
    ADD CONSTRAINT join_requests_status_check
        CHECK (status IN ('requested','accepted','declined','withdrawn'));
-- +goose StatementEnd