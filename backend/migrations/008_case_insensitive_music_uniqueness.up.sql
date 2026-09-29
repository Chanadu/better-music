ALTER TABLE artists
DROP CONSTRAINT artists_user_id_name_key;

CREATE UNIQUE INDEX artists_user_id_lower_name_key
ON artists (user_id, LOWER(name));

ALTER TABLE albums
DROP CONSTRAINT albums_user_id_artist_id_title_key;

CREATE UNIQUE INDEX albums_user_id_artist_id_lower_title_key
ON albums (user_id, artist_id, LOWER(title));
