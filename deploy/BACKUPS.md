# PostgreSQL backups on the Pi

The timer runs daily at **03:00 in the Pi's timezone**, and catches up after
downtime. It retains 30 days of successful custom-format PostgreSQL dumps in
`/var/lib/better-music-backups`, accessible only to postgres and root.
This captures all database tables, including accounts, library data, and migration
metadata. Browser edits that have not synced are not in the server database.

Backups default to a **1 GiB total file-size cap** (including the new dump) and
a **10 GiB free-space reserve**. Before dumping, the script checks database size
and available space, removing oldest backups if necessary while preserving the
latest successful backup. It skips with an error if there is still insufficient
room. A file-size limit bounds the new dump even if the estimate is too small.
Space-driven cleanup can remove older backups before a new backup succeeds.
Concurrent writes by PostgreSQL or other processes can still reduce free space;
this reserve is a backup budget, not a filesystem quota or continuous monitor.
If existing backups already exceed the cap, the last successful backup is kept
and the job fails until there is enough room within the configured limits.

## Install

Run from the repository root on the Pi. Confirm the database name from the
backend's `POSTGRES_URL` first; do not paste credentials into logs or commands.

```sh
sudo install -m 0755 deploy/backup.py /usr/local/bin/better-music-backup
sudo install -m 0644 deploy/better-music-backup.service deploy/better-music-backup.timer /etc/systemd/system/
```

The default database name is `bettermusic`. To override settings, create
`/etc/better-music-backup.conf`, for example:

```ini
DATABASE_NAME=bettermusic
RETENTION_DAYS=30
MAX_BACKUP_MB=1024
MIN_FREE_MB=10240
```

Size settings use MiB (1,048,576 bytes). These files assume a local
PostgreSQL server with the usual postgres peer authentication and socket path.
The script uses Python 3.9+ and its standard library; no pip packages are needed.
If replacing the Bash version, repeat the script install command above. The
installed path, systemd timer, and existing configuration remain the same.

```sh
sudo systemctl daemon-reload
sudo systemctl start better-music-backup.service
sudo journalctl -u better-music-backup.service -n 20 --no-pager
sudo systemctl enable --now better-music-backup.timer
systemctl list-timers better-music-backup.timer
```

Only enable the timer once the first backup succeeds. Inspect failures with
`systemctl status better-music-backup.service` and the journal command above.
The job has no email or push notifications configured.

## Verify a restore

Archive listing is checked on every backup, but a successful restore into an
empty test database is the stronger check. Substitute the exact backup filename:

```sh
sudo -u postgres createdb better_music_restore_check
sudo -u postgres pg_restore --exit-on-error --no-owner --no-privileges --dbname=better_music_restore_check /var/lib/better-music-backups/BACKUP_FILENAME.dump
sudo -u postgres psql -d better_music_restore_check -c 'SELECT count(*) FROM users; SELECT count(*) FROM artists; SELECT count(*) FROM albums;'
sudo -u postgres dropdb better_music_restore_check
```

Use an unused test database name. Never restore into the live database for this
check. For disaster recovery, provision the app's PostgreSQL login and an empty
database owned by that login, then restore with `--no-owner --no-privileges`
as the app login (or use `--role` as a database administrator). Dumps do not
include cluster-wide roles or their passwords. Keep the backend environment
configuration in a separate secure backup/password manager.

## Keep a copy off the Pi

A backup on the Pi's own disk does not survive disk failure or loss of the Pi.
Copy these dumps to a separate computer, NAS, or encrypted remote storage.
Choose the destination and access method before configuring that transfer;
database dumps contain sensitive account data. Keep remote retention independent
of local cleanup so an accidental local deletion does not remove every copy.
