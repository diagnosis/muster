-- +goose Up
-- +goose StatementBegin
ALTER TABLE outings
    ADD COLUMN host_seats INT NOT NULL DEFAULT 0 CHECK (host_seats >= 0),
    ADD COLUMN cost_per_seat_cents INT NOT NULL DEFAULT 0 CHECK (cost_per_seat_cents >= 0);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE outings DROP COLUMN max_size;
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE join_requests
    ADD COLUMN role TEXT NOT NULL DEFAULT 'rider' CHECK (role IN ('rider','driver')),
    ADD COLUMN seats_offered INT NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_role_seats CHECK (
        (role = 'rider'  AND seats_offered = 0) OR
        (role = 'driver' AND seats_offered >= 1)
    );
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE join_requests
DROP CONSTRAINT chk_role_seats,
    DROP COLUMN seats_offered,
    DROP COLUMN role;
-- +goose StatementEnd

-- +goose StatementBegin
-- Lossy: original max_size values are gone; restored rows get the default.
ALTER TABLE outings ADD COLUMN max_size INT NOT NULL DEFAULT 2 CHECK (max_size >= 2);
-- +goose StatementEnd

-- +goose StatementBegin
ALTER TABLE outings
DROP COLUMN cost_per_seat_cents,
    DROP COLUMN host_seats;
-- +goose StatementEnd