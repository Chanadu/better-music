#!/usr/bin/env bash
set -euo pipefail
umask 077

# Run as postgres for local peer authentication; no database password is needed.
DATABASE_NAME="${DATABASE_NAME:-bettermusic}"
BACKUP_DIR="${BACKUP_DIR:-/var/lib/better-music-backups}"
RETENTION_DAYS="${RETENTION_DAYS:-30}"
MAX_BACKUP_MB="${MAX_BACKUP_MB:-1024}"
MIN_FREE_MB="${MIN_FREE_MB:-10240}"
for setting in RETENTION_DAYS MAX_BACKUP_MB MIN_FREE_MB; do
    [[ "${!setting}" =~ ^[1-9][0-9]{0,6}$ ]] || {
        echo "$setting must be a positive integer of at most seven digits" >&2
        exit 1
    }
done
mkdir -p "$BACKUP_DIR"
exec 9>"$BACKUP_DIR/.backup.lock"
flock -n 9 || { echo 'Another backup is running' >&2; exit 1; }

# Allow space for a dump as large as the database, plus 16 MiB of overhead.
# Keep the most recent successful backup even when space is tight.
database_bytes="$(psql --host=/var/run/postgresql --username=postgres \
    --dbname="$DATABASE_NAME" --no-psqlrc --tuples-only --no-align \
    --set=ON_ERROR_STOP=1 --command='SELECT pg_database_size(current_database())')"
[[ "$database_bytes" =~ ^[0-9]+$ ]] || { echo 'Invalid database size' >&2; exit 1; }
required_bytes=$((database_bytes + 16 * 1024 * 1024))
max_bytes=$((MAX_BACKUP_MB * 1024 * 1024))
reserve_bytes=$((MIN_FREE_MB * 1024 * 1024))
mapfile -d '' backups < <(find "$BACKUP_DIR" -maxdepth 1 -type f \
    -name 'better-music-*.dump' -printf '%T@ %p\0' | sort -z -n)
total_bytes=0
for entry in "${backups[@]}"; do
    total_bytes=$((total_bytes + $(stat -c %s -- "${entry#* }")))
done
available_bytes() {
    df -B1 --output=avail "$BACKUP_DIR" | tail -n 1 | tr -d ' '
}
free_bytes="$(available_bytes)"
for ((index=0; index < ${#backups[@]} - 1; index++)); do
    if (( total_bytes + required_bytes <= max_bytes && free_bytes - required_bytes >= reserve_bytes )); then
        break
    fi
    oldest="${backups[index]#* }"
    size="$(stat -c %s -- "$oldest")"
    rm -- "$oldest"
    total_bytes=$((total_bytes - size))
    free_bytes="$(available_bytes)"
done
budget_bytes=$((max_bytes - total_bytes))
free_budget=$((free_bytes - reserve_bytes))
(( free_budget >= budget_bytes )) || budget_bytes=$free_budget
if (( budget_bytes < required_bytes )); then
    echo 'Backup skipped: insufficient room within backup cap or free-space reserve; latest backup preserved' >&2
    exit 1
fi

temporary="$(mktemp "$BACKUP_DIR/.incomplete-XXXXXXXX.dump")"
trap 'rm -f -- "$temporary"' EXIT
# Bound the dump's growth even if the size estimate is too small.
(
    ulimit -c 0
    ulimit -f "$((budget_bytes / 1024))"
    exec pg_dump --host=/var/run/postgresql --username=postgres --dbname="$DATABASE_NAME" \
        --format=custom --file="$temporary"
)
# Check the archive directory before publishing it or pruning older backups.
pg_restore --list "$temporary" >/dev/null
destination="$BACKUP_DIR/better-music-$(date -u +%Y%m%dT%H%M%S)-${temporary##*.incomplete-}"
mv -- "$temporary" "$destination"
find "$BACKUP_DIR" -maxdepth 1 -type f -name 'better-music-*.dump' \
    -mmin "+$((RETENTION_DAYS * 1440))" -delete
echo "Backup saved: $destination"
