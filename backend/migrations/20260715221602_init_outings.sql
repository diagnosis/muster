-- +goose Up
-- +goose StatementBegin
CREATE TABLE outings(
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        host_id     UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
        title       TEXT NOT NULL,
        destination TEXT NOT NULL,               -- trail-as-text until Trail returns (v1+)
        meet_label  TEXT NOT NULL,               -- "Upper parking lot, Rattlesnake Lake"
        meet_lat    DOUBLE PRECISION,
        meet_lng    DOUBLE PRECISION,
        starts_at   TIMESTAMPTZ NOT NULL,
        max_size    INT NOT NULL CHECK (max_size >= 2),   -- TOTAL headcount, host included
        difficulty  TEXT NOT NULL CHECK (difficulty IN ('easy','moderate','hard')),
        pace        TEXT NOT NULL CHECK (pace IN ('relaxed','moderate','fast')),  -- THE pace
        notes       TEXT,
        status      TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open','cancelled')),
        created_at  TIMESTAMPTZ DEFAULT now(),
        updated_at  TIMESTAMPTZ DEFAULT now()
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE join_requests (
        id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        outing_id  UUID NOT NULL REFERENCES outings(id) ON DELETE CASCADE,
        hiker_id   UUID NOT NULL REFERENCES hikers(id) ON DELETE CASCADE,
        status     TEXT NOT NULL DEFAULT 'requested'
            CHECK (status IN ('requested','accepted','declined','withdrawn')),
        note       TEXT,
        created_at TIMESTAMPTZ DEFAULT now(),
        updated_at TIMESTAMPTZ DEFAULT now(),
        CONSTRAINT uq_outing_hiker UNIQUE (outing_id, hiker_id)
);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS join_requests;
-- +goose StatementEnd
    
-- +goose StatementBegin
DROP TABLE IF EXISTS outings;
-- +goose StatementEnd


