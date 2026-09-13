-- +goose Up
-- +goose StatementBegin
CREATE TABLE comments (
        id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        outing_id  UUID NOT NULL REFERENCES outings(id) ON DELETE CASCADE,
        hiker_id   UUID NOT NULL REFERENCES hikers(id)  ON DELETE CASCADE,
        parent_id  UUID NULL REFERENCES comments(id) ON DELETE CASCADE,   -- null = top-level; one level deep, enforced in service
        body       TEXT NOT NULL,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        deleted_at TIMESTAMPTZ NULL                                       -- soft-delete; replies survive, parent renders as stub
);

CREATE INDEX idx_comments_outing_created ON comments (outing_id, created_at);

CREATE TABLE comment_likes (
        comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
        hiker_id   UUID NOT NULL REFERENCES hikers(id)   ON DELETE CASCADE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
        PRIMARY KEY (comment_id, hiker_id)                                 -- one like per hiker per comment
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS comment_likes;
DROP TABLE IF EXISTS comments;
-- +goose StatementEnd