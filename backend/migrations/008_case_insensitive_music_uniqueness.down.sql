DROP INDEX albums_user_id_artist_id_lower_title_key;

ALTER TABLE albums
ADD CONSTRAINT albums_user_id_artist_id_title_key UNIQUE (user_id, artist_id, title);

DROP INDEX artists_user_id_lower_name_key;

ALTER TABLE artists
ADD CONSTRAINT artists_user_id_name_key UNIQUE (user_id, name);
