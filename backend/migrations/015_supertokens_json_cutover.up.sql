BEGIN;

-- Export legacy libraries before this cutover. They transfer through JSON.
DROP TABLE refresh_tokens, processed_mutations, albums, artists;
TRUNCATE TABLE library_albums, library_artists, users;

ALTER TABLE library_albums DROP CONSTRAINT library_albums_user_id_artist_id_fkey;
ALTER TABLE library_albums DROP CONSTRAINT library_albums_user_id_fkey;
ALTER TABLE library_artists DROP CONSTRAINT library_artists_user_id_fkey;

ALTER TABLE users
    ALTER COLUMN id DROP DEFAULT,
    ALTER COLUMN id TYPE TEXT,
    DROP COLUMN password_hash,
    ADD COLUMN auth_disabled BOOLEAN NOT NULL DEFAULT false;
DROP SEQUENCE users_id_seq;

ALTER TABLE library_artists ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE library_albums ALTER COLUMN user_id TYPE TEXT;
ALTER TABLE library_artists ADD FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE library_albums ADD FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE;
ALTER TABLE library_albums ADD FOREIGN KEY (user_id, artist_id)
    REFERENCES library_artists(user_id, id) DEFERRABLE INITIALLY DEFERRED;

CREATE TABLE auth_account_operations (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE,
    kind TEXT NOT NULL CHECK (kind IN ('email', 'delete')),
    target_email TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (
        (kind = 'email' AND target_email IS NOT NULL)
        OR (kind = 'delete' AND target_email IS NULL)
    )
);

CREATE UNIQUE INDEX auth_pending_email
    ON auth_account_operations (lower(target_email))
    WHERE target_email IS NOT NULL;

COMMIT;
