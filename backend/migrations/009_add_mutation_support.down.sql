BEGIN;

DROP TABLE processed_mutations;
ALTER TABLE albums DROP COLUMN version;
ALTER TABLE artists DROP COLUMN version;

COMMIT;
