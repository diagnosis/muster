-- +goose Up
-- +goose StatementBegin
CREATE table auth_tokens(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hiker_id UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    purpose TEXT NOT NULL check (purpose IN ('email_verification', 'forgot_password', 'login_code' )),
    created_at TIMESTAMPTZ DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ
);
ALTER TABLE hikers ADD column verified_at TIMESTAMPTZ;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_tokens;
ALTER TABLE hikers DROP column verified_at;
-- +goose StatementEnd
