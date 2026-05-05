ALTER DATABASE app_dbs SET intervalstyle TO 'iso_8601';




CREATE TABLE IF NOT EXISTS room (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    deleted_by TEXT,
    deleted_at TIMESTAMPTZ
);

-- SELECT * FROM room;

CREATE OR REPLACE VIEW active_room AS
SELECT id FROM room
WHERE deleted_at IS NULL;


-- create new room and return its id
INSERT INTO room (created_by) VALUES ('Bob') RETURNING id;


-- list all active rooms
SELECT id FROM active_room;


-- delete a room by setting deleted_at and deleted_by
UPDATE room
SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, 'alice')
WHERE id = '573defb2-1a79-4532-bf71-1b2e625c2ef6';





CREATE TABLE IF NOT EXISTS series (
    id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
    room_id uuid REFERENCES room(id) ON DELETE CASCADE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    name TEXT NOT NULL,
    description TEXT,
    color TEXT,
    tag_list TEXT[],
    target_interval INTERVAL,
    meta JSON,

    deleted_at TIMESTAMPTZ,
    deleted_by TEXT
);

-- SELECT * FROM series;

CREATE OR REPLACE VIEW active_series AS
SELECT
    s.id,
    s.room_id,
    s.created_at,
    s.created_by,
    s.name,
    s.description,
    s.color,
    s.tag_list,
    s.target_interval::text,
    s.meta,
    e.time_diffs,
    e.median_diff::text,
    e.last_date,
    e.last_date + s.target_interval AS next_target_date,
    e.last_date + e.median_diff AS next_median_date
FROM series s
LEFT JOIN LATERAL (
    SELECT
        array_agg(trunc(EXTRACT(EPOCH FROM x.time_diff) * 1000)) AS time_diffs,
        MAX(x.created_at) AS last_date,
        PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY x.time_diff) AS median_diff
    FROM (
        SELECT
            created_at,
            created_at - LAG(created_at) OVER (ORDER BY created_at) AS time_diff
        FROM event
        WHERE series_id = s.id
    ) x
) e ON TRUE
WHERE s.deleted_at IS NULL;

-- SELECT * FROM active_series;


-- create new series with all mandatory fields and return its id
INSERT INTO series
    (room_id, created_by, name, description, color, tag_list, target_interval, meta)
VALUES
    ('3b3a243a-92f0-4790-9474-11743e2bf1d1', 'Bob', 'Series 1', NULL, NULL, NULL, NULL, NULL)
RETURNING id;


-- create new series with all fields and return its id
INSERT INTO series
    (room_id, created_by, name, description, color, tag_list, target_interval, meta)
VALUES
    ('3b3a243a-92f0-4790-9474-11743e2bf1d1', 'Bob', 'Series 2', 'This is a description for Series 3', '#ff0000', ARRAY['tag3'], 'P3DT8H', '{"key": "value"}'::json) -- 'P3DT8H' is ISO 8601 duration format for 3 days and 8 hours
RETURNING id;


-- list all active series in a specific room
SELECT id, created_at, created_by, name, description, color, tag_list, target_interval, meta, time_diffs, median_diff, last_date, next_target_date, next_median_date FROM active_series WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1';


-- get a specific series by id
SELECT id, created_at, created_by, name, description, color, tag_list, target_interval, meta, time_diffs, median_diff, last_date, next_target_date, next_median_date FROM active_series WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id = '02b096d3-bea7-49a8-b551-fcfa867d882d';


-- delete a series by setting deleted_at and deleted_by
UPDATE series
SET deleted_at = COALESCE(deleted_at, NOW()), deleted_by = COALESCE(deleted_by, 'JimBob')
WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id = '1660473a-5864-44a3-b25a-6defdf49f0cd';


-- update series with all fields
UPDATE series
SET name = 'Updated Series Name updated', description = 'Updated description', color = '#00ff00', tag_list = ARRAY['tag1', 'tag2'], target_interval = 'P1DT12H', meta = '{"updated_key": "updated_value"}'::json
WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id = '02b096d3-bea7-49a8-b551-fcfa867d882d';


-- update series with mandatory fields only
UPDATE series
SET name = 'new name', description = NULL, color = NULL, tag_list = NULL, target_interval = NULL, meta = NULL
WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id = '02b096d3-bea7-49a8-b551-fcfa867d882d';





CREATE TABLE IF NOT EXISTS event (
    id SERIAL PRIMARY KEY,
    series_id uuid REFERENCES series(id) ON DELETE CASCADE,
    created_by TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    meta JSON
);

-- SELECT * FROM event;


-- create new event with all mandatory fields and return the created event
INSERT INTO event (series_id, created_by, created_at, meta) VALUES
    ('02b096d3-bea7-49a8-b551-fcfa867d882d', 'eve', DEFAULT, NULL)
RETURNING id, created_by, created_at, meta;


-- create new event with all optional fields and return the created event
INSERT INTO event (series_id, created_by, created_at, meta) VALUES
    ('02b096d3-bea7-49a8-b551-fcfa867d882d', 'eve', '2025-10-01T14:00:00+01:00', '{"event_key": "event_value"}'::json)
RETURNING id, created_by, created_at, meta;


-- list all events for a specific series
SELECT id, created_at, created_by, meta, (created_at - LAG(created_at) OVER (ORDER BY created_at))::text AS time_diff FROM event WHERE series_id = '02b096d3-bea7-49a8-b551-fcfa867d882d';


-- delete an event
DELETE FROM event WHERE series_id = '02b096d3-bea7-49a8-b551-fcfa867d882d' AND id = 8;


-- update an event
UPDATE event
SET created_at = '2025-10-01T01:00:00+01:00', meta = '{"updated_event_key": "updated_event_value"}'::json
WHERE series_id = '02b096d3-bea7-49a8-b551-fcfa867d882d' AND id = 3
RETURNING id, created_by, created_at, meta;





-- get an iCalendar feed for specific series
SELECT id, name, description, last_date, next_target_date, next_median_date
FROM active_series
WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id IN ('1660473a-5864-44a3-b25a-6defdf49f0cd', '02b096d3-bea7-49a8-b551-fcfa867d882d');

SELECT id, name, description, last_date, next_target_date, next_median_date
FROM active_series
WHERE room_id = '3b3a243a-92f0-4790-9474-11743e2bf1d1' AND id::text = ANY(ARRAY['1660473a-5864-44a3-b25a-6defdf49f0cd', '02b096d3-bea7-49a8-b551-fcfa867d882d']);