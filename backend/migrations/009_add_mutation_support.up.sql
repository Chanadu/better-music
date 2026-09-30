BEGIN;

ALTER TABLE artists
ADD COLUMN version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0);

ALTER TABLE albums
ADD COLUMN version INTEGER NOT NULL DEFAULT 1 CHECK (version > 0);

-- Keys belong to an account, so different users may reuse the same key.
-- No expiry: these responses are retained until the account is deleted.
CREATE TABLE processed_mutations (
	user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	mutation_id TEXT NOT NULL CHECK (length(mutation_id) > 0),
	request_hash TEXT NOT NULL CHECK (length(request_hash) > 0),
	response_status INTEGER NOT NULL CHECK (response_status BETWEEN 100 AND 599),
	response_headers JSONB NOT NULL DEFAULT '{}'::jsonb
		CHECK (jsonb_typeof(response_headers) = 'object'),
	response_body BYTEA NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	PRIMARY KEY (user_id, mutation_id)
);

COMMIT;
