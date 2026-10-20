# Supabase migration

The original app still uses its old database. Supabase contains the imported copy: **84 artists, 199 albums**, with ratings **1–10**. Do not rerun the import for that account.

| Location | Purpose |
| --- | --- |
| `migrations/` | Database changes, applied once in filename order |
| `scripts/import_library.py` | Validate an export and generate rehearsal or final SQL |
| `tests/` | Checks against a disposable local database |
| `.local/` | Personal exports and generated SQL; ignored by Git |

Keep applied migrations unchanged. The temporary 0–10 migration is followed by the migration restoring 1–10. Manual Studio execution does not register Supabase CLI migration history.

## Import tool

Run from the repository root. Replace `export.json` and `USER_UUID` below.

```bash
# Validate and show counts. No files or database writes.
python3 supabase/scripts/import_library.py export.json --user-id USER_UUID

# Generate a rehearsal that rolls back.
python3 supabase/scripts/import_library.py export.json --user-id USER_UUID \
  --output supabase/.local/rehearsal.sql

# Generate a final import using the same checks, ending with commit.
python3 supabase/scripts/import_library.py export.json --user-id USER_UUID \
  --output supabase/.local/import.sql --save
```

The tool never connects to a database. Execute generated SQL by pasting the **entire file** into the new Supabase Studio. It requires an existing account and an empty destination, imports artists before albums, and verifies all values except the server sync timestamp. Output files are private and cannot overwrite existing files.

Existing `.local/` files are preserved. The approved import copy changes only the three original zero ratings to one; validation never silently changes values. IDs stay deterministic. Reconcile old-device queues and use a fresh export if the source changes. Exports do not back up passwords or the full offline queue.

## Checks

```bash
python3 -m unittest discover -s supabase/tests -p 'test_*.py'
python3 supabase/tests/verify_library.py

# Also test rehearsal, saved import, and duplicate-import refusal locally:
python3 supabase/tests/verify_library.py --export export.json --user-id USER_UUID
```

Database checks require PostgreSQL binaries and stub Supabase Auth. They never connect to existing databases.

## Next: real frontend

The development-only `/migration` page downloads Supabase records through RxDB; it cannot upload edits. Configure `frontend/.env.local` using `.env.example`, then start the frontend dev server. Verify **84 artists / 199 albums / 159 listened / 40 queued**, including familiar ratings and notes. The page returns 404 in production.

Before enabling edits, test auth expiry, clock differences, artist/album write order, deletion behavior, and duplicate-name recovery. See `../MIGRATION-PLAN.md` for the full rollout plan.
