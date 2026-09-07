-- +goose Up
-- +goose StatementBegin
CREATE TABLE notification_events(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    hiker_id UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK(kind IN ('join_request_created', 'join_request_approved', 'join_request_declined', 'join_request_withdrawn', 'member_removed', 'outing_cancelled', 'outing_updated')),
    payload jsonb NOT NULL ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    read_at TIMESTAMPTZ NULL,
    emailed_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_notification_events_unsent ON notification_events (created_at) WHERE emailed_at IS NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS notification_events;
-- +goose StatementEnd
