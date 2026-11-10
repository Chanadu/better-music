BEGIN;

ALTER TABLE users
    ADD COLUMN library_sync_sequence BIGINT NOT NULL DEFAULT 0 CHECK (library_sync_sequence >= 0),
    ADD COLUMN library_sync_epoch UUID NOT NULL DEFAULT gen_random_uuid();

CREATE TABLE library_artists (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (length(btrim(name)) > 0),
    cover_url TEXT,
    spotify_id TEXT,
    created_at TEXT NOT NULL,
    edited_at TIMESTAMPTZ NOT NULL,
    _deleted BOOLEAN NOT NULL DEFAULT false,
    change_seq BIGINT NOT NULL CHECK (change_seq > 0),
    UNIQUE(user_id,id)
);

CREATE TABLE library_albums (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    artist_id UUID NOT NULL,
    title TEXT NOT NULL CHECK (length(btrim(title)) > 0),
    cover_url TEXT,
    spotify_id TEXT,
    year INTEGER,
    listened BOOLEAN NOT NULL,
    rating INTEGER CHECK (rating BETWEEN 1 AND 10),
    comment TEXT,
    listened_at TEXT,
    created_at TEXT NOT NULL,
    edited_at TIMESTAMPTZ NOT NULL,
    _deleted BOOLEAN NOT NULL DEFAULT false,
    change_seq BIGINT NOT NULL CHECK (change_seq > 0),
    FOREIGN KEY(user_id,artist_id) REFERENCES library_artists(user_id,id),
    CHECK (listened OR (rating IS NULL AND listened_at IS NULL))
);

CREATE UNIQUE INDEX library_artists_names ON library_artists(user_id,lower(name)) WHERE NOT _deleted;
CREATE UNIQUE INDEX library_albums_titles ON library_albums(user_id,artist_id,lower(title)) WHERE NOT _deleted;
CREATE INDEX library_artists_pull ON library_artists(user_id,change_seq);
CREATE INDEX library_albums_pull ON library_albums(user_id,change_seq);

COMMIT;
