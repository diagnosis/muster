-- +goose Up
-- +goose StatementBegin
    CREATE TABLE conversation_reads(
        conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
        hiker_id UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
        last_read_seq BIGINT NOT NULL,
        updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        PRIMARY KEY (conversation_id, hiker_id)
    );
    -- hosts: one marker per outing conversation
    INSERT INTO conversation_reads (conversation_id, hiker_id, last_read_seq)
    SELECT c.id,
           o.host_id,
           COALESCE((SELECT MAX(m.seq) FROM messages m WHERE m.conversation_id = c.id), 0)
    FROM conversations c
             JOIN outings o ON o.id = c.outing_id
    WHERE c.kind = 'outing';

    INSERT INTO conversation_reads(conversation_id, hiker_id, last_read_seq)
    SELECT c.id, jr.hiker_id,
           COALESCE((SELECT MAX(m.seq) FROM messages m WHERE m.conversation_id = c.id), 0)
    FROM conversations c
            JOIN join_requests jr on c.outing_id = jr.outing_id
    WHERE c.kind = 'outing' AND jr.status = 'accepted'
        ON CONFLICT DO NOTHING;

    INSERT INTO conversation_reads(conversation_id, hiker_id, last_read_seq)
    SELECT c.id, c.dm_a,
           COALESCE((SELECT MAX(m.seq) FROM messages m WHERE m.conversation_id = c.id), 0)
           FROM conversations c WHERE kind = 'dm';

    INSERT INTO conversation_reads(conversation_id, hiker_id, last_read_seq)
    SELECT c.id, c.dm_b,
           COALESCE((SELECT MAX(m.seq) FROM messages m WHERE m.conversation_id = c.id), 0)
    FROM conversations c WHERE kind = 'dm';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS conversation_reads;
-- +goose StatementEnd
