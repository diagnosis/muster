-- +goose Up
-- +goose StatementBegin
CREATE TABLE conversations
(
    id             UUID PRIMARY KEY     DEFAULT gen_random_uuid(),
    kind           TEXT        NOT NULL CHECK (kind IN ('outing', 'dm')),
    outing_id      UUID UNIQUE NULL REFERENCES outings(id) ON DELETE CASCADE,
    dm_a           UUID NULL REFERENCES hikers(id) ON DELETE CASCADE,
    dm_b           UUID NULL REFERENCES hikers(id) ON DELETE CASCADE,
    dm_initiator UUID NULL REFERENCES hikers(id),
    dm_status      TEXT CHECK (dm_status IN ('pending', 'accepted', 'declined')),
    dm_declined_by UUID NULL REFERENCES hikers(id) CHECK( (dm_status = 'declined') = (dm_declined_by IS NOT NULL) ),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT conversations_kind_shape CHECK (
        (kind = 'outing'
            AND outing_id IS NOT NULL
            AND dm_a IS NULL AND dm_b IS NULL
            AND dm_initiator IS NULL AND dm_status IS NULL)
            OR (kind = 'dm'
            AND outing_id IS NULL
            AND dm_a IS NOT NULL AND dm_b IS NOT NULL AND dm_a < dm_b
            AND dm_initiator IS NOT NULL AND dm_initiator IN (dm_a, dm_b)
            AND dm_status IS NOT NULL)
        ),
    UNIQUE (dm_a, dm_b)
);

CREATE TABLE messages
(
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE ,
    hiker_id UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
    seq BIGINT GENERATED ALWAYS AS IDENTITY UNIQUE ,
    body TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE INDEX idx_messages_conversation_id_seq ON messages (conversation_id, seq);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;
-- +goose StatementEnd
