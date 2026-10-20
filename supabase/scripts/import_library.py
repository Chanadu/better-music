"""Validate an export and optionally generate SQL. Never connects to a database."""

import argparse
import hashlib
import json
import os
import uuid
from datetime import datetime
from pathlib import Path


FIELDS = {
    'artists': ('id', 'name', 'cover_url', 'spotify_id', 'created_at'),
    'albums': (
        'id',
        'artist_id',
        'title',
        'cover_url',
        'year',
        'spotify_id',
        'listened',
        'rating',
        'comment',
        'listened_at',
        'created_at',
    ),
}


def load_export(path, user_id):
    """Validate once, then map old IDs to stable per-account UUIDs."""
    owner = str(uuid.UUID(user_id))
    raw = Path(path).read_bytes()
    export = json.loads(raw)

    if (
        not isinstance(export, dict)
        or export.get('format_version') != 2
        or export.get('application') != 'better-music'
    ):
        raise ValueError('Expected a Better Music format-version 2 export.')

    library = export.get('library')

    if not isinstance(library, dict) or any(
        not isinstance(library.get(table), list) for table in FIELDS
    ):
        raise ValueError('Export must contain artist and album arrays.')

    def mapped_id(table, old_id):
        return str(uuid.uuid5(uuid.UUID(owner), f'better-music-v1:{table}:{old_id}'))

    records = {'artists': [], 'albums': []}
    artist_ids = set()

    for table, fields in FIELDS.items():
        ids, names = set(), set()

        for index, original in enumerate(library[table]):
            label = f'{table}[{index}]'

            def require(valid, message):
                if not valid:
                    raise ValueError(f'{label}: {message}')

            require(
                isinstance(original, dict) and set(original) == set(fields),
                'unexpected or missing fields',
            )

            row = dict(original)
            old_id = row['id']

            require(
                type(old_id) is int and old_id > 0,
                'expected positive server ID; reconcile offline records first',
            )
            require(old_id not in ids, 'duplicate ID')
            ids.add(old_id)

            title = row['name' if table == 'artists' else 'title']
            require(isinstance(title, str) and bool(title.strip()), 'name/title is blank')

            for field in ('cover_url', 'spotify_id') + (('comment',) if table == 'albums' else ()):
                require(row[field] is None or isinstance(row[field], str), f'invalid {field}')

            for field in ('created_at',) + (('listened_at',) if table == 'albums' else ()):
                value = row[field]

                if field == 'listened_at' and value is None:
                    continue

                try:
                    require(isinstance(value, str), f'invalid {field}')
                    timestamp = datetime.fromisoformat(value.replace('Z', '+00:00'))
                    require(timestamp.tzinfo is not None, f'{field} needs an explicit timezone')
                except (TypeError, ValueError) as error:
                    raise ValueError(f'{label}: invalid or timezone-less {field}') from error

            if table == 'artists':
                artist_ids.add(old_id)
                name = title.lower()
            else:
                artist_id = row['artist_id']
                require(
                    type(artist_id) is int and artist_id in artist_ids,
                    'artist reference is missing',
                )
                require(type(row['listened']) is bool, 'listened must be true or false')

                require(
                    row['rating'] is None
                    or (type(row['rating']) is int and 1 <= row['rating'] <= 10),
                    'rating must be 1–10 or null',
                )

                require(
                    row['year'] is None
                    or (
                        type(row['year']) is int
                        and -2147483648 <= row['year'] <= 2147483647
                    ),
                    'invalid year',
                )

                require(
                    row['listened']
                    or (row['rating'] is None and row['listened_at'] is None),
                    'queued album has listening metadata',
                )

                name = (artist_id, title.lower())
                row['artist_id'] = mapped_id('artists', artist_id)

            require(name not in names, 'duplicate case-insensitive name/title')
            names.add(name)

            row.update(
                id=mapped_id(table, old_id),
                user_id=owner,
                _deleted=False,
                last_edit_ms=0,
                edit_id='',
            )
            records[table].append(row)

    report = {
        'user_id': owner,
        'source_sha256': hashlib.sha256(raw).hexdigest(),
        'artists': len(records['artists']),
        'albums': len(records['albums']),
        'listened': sum(row['listened'] for row in records['albums']),
        'queued': sum(not row['listened'] for row in records['albums']),
    }

    return records, report


def make_sql(records, report, save=False):
    """The same checked transaction is used for rehearsal and final import."""
    owner = report['user_id']

    sql = f"""-- {'FINAL IMPORT: saves records' if save else 'REHEARSAL: rolls back all records'}
-- Run the entire file in the NEW Supabase Studio.
-- Source SHA256: {report['source_sha256']}
begin;

do $$
begin
  if not exists (select 1 from auth.users where id = '{owner}'::uuid) then
    raise exception 'Target Supabase account does not exist';
  end if;
  if exists (select 1 from public.artists where user_id = '{owner}'::uuid)
     or exists (select 1 from public.albums where user_id = '{owner}'::uuid) then
    raise exception 'Target library is not empty. Stop and inspect it.';
  end if;
end;
$$;
"""

    for table, fields in FIELDS.items():
        payload = json.dumps(records[table], ensure_ascii=False, indent=2)
        delimiter = '$import_' + uuid.uuid4().hex + '$'
        columns = ', '.join((*fields, 'user_id', '_deleted', 'last_edit_ms', 'edit_id'))

        sql += f"""
create temp table expected_{table} on commit drop as
select * from jsonb_populate_recordset(null::public.{table},
{delimiter}
{payload}
{delimiter}::jsonb);

insert into public.{table} ({columns})
select {columns} from expected_{table};

do $$
begin
  if exists (
    select 1 from expected_{table} as expected
    full join (
      select * from public.{table} where user_id = '{owner}'::uuid
    ) as actual using (id)
    where (to_jsonb(expected) - '_modified')
      is distinct from (to_jsonb(actual) - '_modified')
  ) then
    raise exception '{table}: imported values differ from the export';
  end if;
end;
$$;
"""

    return sql + f"""
{'commit' if save else 'rollback'};

select
  '{'Import complete; records saved' if save else 'Rehearsal passed; no records saved'}' as result,
  {report['artists']} as artists_checked,
  {report['albums']} as albums_checked;
"""


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('export', type=Path)
    parser.add_argument('--user-id', required=True)
    parser.add_argument(
        '--output',
        type=Path,
        help='Write rehearsal SQL; otherwise just preview counts',
    )
    parser.add_argument(
        '--save',
        action='store_true',
        help='Generate committing SQL instead of a rehearsal',
    )

    args = parser.parse_args()

    if args.save and not args.output:
        parser.error('--save requires --output')

    try:
        records, report = load_export(args.export, args.user_id)

        if args.output:
            with open(
                args.output,
                'x',
                opener=lambda path, flags: os.open(path, flags, 0o600),
            ) as file:
                file.write(make_sql(records, report, save=args.save))

        print(json.dumps(report, indent=2))

        if args.output:
            print(f"SQL generated ({'saves records' if args.save else 'rollback only'}): {args.output}")

    except (ValueError, KeyError, OSError) as error:
        parser.exit(1, f'Import preparation failed: {error}\n')


if __name__ == '__main__':
    main()
