-- +goose Up
-- +goose StatementBegin
ALTER TABLE notification_events DROP CONSTRAINT notification_events_kind_check;
ALTER TABLE notification_events ADD CONSTRAINT notification_events_kind_check
    CHECK (kind IN ('join_request_created', 'join_request_approved', 'join_request_declined',
                    'join_request_withdrawn', 'member_removed', 'outing_cancelled', 'outing_updated',
                    'dm_requested', 'dm_accepted', 'dm_reopened'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE notification_events DROP CONSTRAINT notification_events_kind_check;
ALTER TABLE notification_events ADD CONSTRAINT notification_events_kind_check
    CHECK (kind IN ('join_request_created', 'join_request_approved', 'join_request_declined',
                    'join_request_withdrawn', 'member_removed', 'outing_cancelled', 'outing_updated'));
-- +goose StatementEnd