#!/usr/bin/python3
"""Back up local PostgreSQL with bounded storage and oldest-first cleanup."""

import fcntl
import os
from pathlib import Path
import resource
import shutil
import subprocess
import sys
import time

MIB = 1024 * 1024


def setting(name: str, default: int) -> int:
    value = int(os.environ.get(name, str(default)))
    if value <= 0:
        raise ValueError(f"{name} must be positive")
    return value


def database_size(connection):
    size = subprocess.check_output(
        ["psql", *connection, "-XAt", "--set=ON_ERROR_STOP=1",
         "--command=SELECT pg_database_size(current_database())"], text=True
    )
    return int(size)


def make_room(directory, backups, required, cap, reserve):
    while True:
        used = sum(p.stat().st_size for p in directory.iterdir() if p.is_file())
        budget = min(cap - used, shutil.disk_usage(directory).free - reserve)
        if budget >= required:
            return budget
        if len(backups) <= 1:
            raise RuntimeError("Backup skipped: storage limit reached; latest backup preserved")
        backups.pop(0).unlink()


def create_backup(directory, connection, budget):
    def limit_dump():
        resource.setrlimit(resource.RLIMIT_CORE, (0, 0))
        resource.setrlimit(resource.RLIMIT_FSIZE, (budget, budget))

    temporary = directory / ".incomplete.dump"
    try:
        with temporary.open("wb") as output:
            subprocess.run(["pg_dump", *connection, "--format=custom"],
                           stdout=output, check=True, preexec_fn=limit_dump)
        subprocess.run(["pg_restore", "--list", str(temporary)], check=True,
                       stdout=subprocess.DEVNULL)
        destination = directory / f"better-music-{time.time_ns()}.dump"
        temporary.rename(destination)
        return destination
    finally:
        temporary.unlink(missing_ok=True)


def remove_expired(backups, retention):
    cutoff = time.time() - retention
    for backup in backups:
        if backup.stat().st_mtime < cutoff:
            backup.unlink()


def main():
    os.umask(0o077)

    directory = Path(os.environ.get("BACKUP_DIR", "/var/lib/better-music-backups"))
    database = os.environ.get("DATABASE_NAME", "bettermusic")
    retention = setting("RETENTION_DAYS", 30) * 86400
    cap = setting("MAX_BACKUP_MB", 1024) * MIB
    reserve = setting("MIN_FREE_MB", 10240) * MIB
    directory.mkdir(parents=True, exist_ok=True)

    with (directory / ".backup.lock").open("a") as lock:
        try:
            fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        except BlockingIOError:
            raise RuntimeError("Another backup is running") from None

        connection = ["--host=/var/run/postgresql", "--username=postgres", f"--dbname={database}"]
        backups = sorted(directory.glob("better-music-*.dump"), key=lambda p: p.stat().st_mtime)
        required = database_size(connection) + 16 * MIB
        budget = make_room(directory, backups, required, cap, reserve)
        destination = create_backup(directory, connection, budget)
        remove_expired(backups, retention)
        print(f"Backup saved: {destination}")


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        sys.exit(f"Backup failed: {error}")
