"""Check schema and optional import in a disposable local PostgreSQL cluster."""

import argparse
import subprocess
import sys
import tempfile
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
sys.path.insert(0, str(ROOT / 'scripts'))
from import_library import load_export, make_sql


def run(*args, **kwargs):
    return subprocess.run(args, check=True, text=True, capture_output=True, **kwargs).stdout


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--export', type=Path)
    parser.add_argument('--user-id')
    args = parser.parse_args()

    if args.export and not args.user_id:
        parser.error('--export requires --user-id')

    prepared = load_export(args.export, args.user_id) if args.export else None

    with tempfile.TemporaryDirectory(prefix='better-music-schema-') as temporary:
        data = str(Path(temporary) / 'data')
        log = Path(temporary) / 'postgres.log'
        started = False

        try:
            run(
                'initdb',
                '-D', data,
                '-A', 'trust',
                '-U', 'postgres',
                '--no-locale',
                '--encoding=UTF8',
            )
            run(
                'pg_ctl',
                '-D', data,
                '-l', str(log),
                '-o', f"-F -k {temporary} -c listen_addresses='' -c wal_level=logical",
                '-w', 'start',
            )
            started = True

            psql = [
                'psql', '-X',
                '-h', temporary,
                '-U', 'postgres',
                '-d', 'postgres',
                '-v', 'ON_ERROR_STOP=1',
            ]

            run(*psql, input="""
create role anon;
create role authenticated;
create schema auth;
create table auth.users(id uuid primary key);
create function auth.uid() returns uuid language sql stable as
  $$ select nullif(current_setting('request.jwt.claim.sub', true), '')::uuid $$;
grant usage on schema auth to authenticated;
grant execute on function auth.uid() to authenticated;
create publication supabase_realtime;
""")

            for migration in sorted((ROOT / 'migrations').glob('*.sql')):
                run(*psql, '-f', str(migration))

            run(*psql, '-f', str(ROOT / 'tests/library.sql'))
            print('Schema, ownership, rating, and deletion checks passed.')

            if prepared:
                records, report = prepared
                owner = report['user_id']

                run(*psql, input=f"insert into auth.users values ('{owner}');")
                counts = f"""select
                  (select count(*) from public.artists where user_id = '{owner}'),
                  (select count(*) from public.albums where user_id = '{owner}');"""

                for save in [False, True]:
                    run(*psql, input=make_sql(records, report, save=save))
                    actual = run(*psql, '-tA', input=counts).strip()
                    expected = f"{report['artists']}|{report['albums']}" if save else '0|0'

                    if actual != expected:
                        raise ValueError(f'Import counts differ: {actual} != {expected}')

                try:
                    run(*psql, input=make_sql(records, report, save=True))
                except subprocess.CalledProcessError as error:
                    if 'Target library is not empty' not in error.stderr:
                        raise
                else:
                    raise ValueError('Repeated import unexpectedly succeeded')

                print('Import rollback, saved values/counts, and duplicate-import protection passed.')

        except subprocess.CalledProcessError as error:
            print(error.stdout or '')
            print(error.stderr or '')

            if log.exists() and not started:
                print(log.read_text())

            raise

        finally:
            if started:
                run('pg_ctl', '-D', data, '-m', 'immediate', '-w', 'stop')


if __name__ == '__main__':
    main()
