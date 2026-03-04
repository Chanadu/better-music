CREATE TABLE artists(
	id SERIAL PRIMARY KEY,
	user_id INTEGER NOT NULL REFERENCES users(id),
	name TEXT NOT NULL,
	spotify_id TEXT,
	created_at TIMESTAMP DEFAULT NOW(),
	UNIQUE(user_id, spotify_id)
);
