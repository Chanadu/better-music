CREATE TABLE albums(
	id SERIAL PRIMARY KEY,
	artist_id INTEGER NOT NULL REFERENCES artists(id),
	user_id INTEGER NOT NULL REFERENCES users(id),
	title TEXT NOT NULL,
	cover_url TEXT,
	YEAR INTEGER,
	spotify_id TEXT NOT NULL,
	listened BOOLEAN DEFAULT FALSE,
	rating INTEGER,
	comment TEXT,
	listened_at TIMESTAMP,
	created_at TIMESTAMP DEFAULT NOW()
);
