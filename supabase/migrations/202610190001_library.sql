begin;

create table public.artists (
  id text primary key check (length(id) between 1 and 100),
  user_id uuid not null references auth.users(id) on delete cascade,

  name text not null check (length(btrim(name)) > 0),
  cover_url text,
  spotify_id text,
  created_at timestamptz not null default now(),

  _deleted boolean not null default false,
  _modified timestamptz not null default clock_timestamp(),

  last_edit_ms bigint not null default 0
    check (last_edit_ms between 0 and 9007199254740991),

  edit_id text not null default ''
    check (length(edit_id) <= 100),

  unique (user_id, id)
);


create table public.albums (
  id text primary key check (length(id) between 1 and 100),
  user_id uuid not null references auth.users(id) on delete cascade,
  artist_id text not null,

  title text not null check (length(btrim(title)) > 0),
  cover_url text,
  year integer,
  spotify_id text,

  listened boolean not null default false,
  rating integer check (rating between 1 and 10),
  comment text,
  listened_at timestamptz,
  created_at timestamptz not null default now(),

  _deleted boolean not null default false,
  _modified timestamptz not null default clock_timestamp(),

  last_edit_ms bigint not null default 0
    check (last_edit_ms between 0 and 9007199254740991),

  edit_id text not null default ''
    check (length(edit_id) <= 100),

  foreign key (user_id, artist_id)
    references public.artists(user_id, id),

  check (listened or (rating is null and listened_at is null))
);


create unique index artists_live_name
  on public.artists(user_id, lower(name))
  where not _deleted;

create unique index albums_live_title
  on public.albums(user_id, artist_id, lower(title))
  where not _deleted;

create index artists_sync_cursor
  on public.artists(user_id, _modified, id);

create index albums_sync_cursor
  on public.albums(user_id, _modified, id);

create index albums_artist
  on public.albums(user_id, artist_id);


create function public.set_library_sync_modified()
returns trigger
language plpgsql
security invoker
set search_path = ''
as $$
begin
  new._modified := clock_timestamp();
  return new;
end;
$$;

revoke all on function public.set_library_sync_modified()
  from public, anon, authenticated;

create trigger artists_sync_modified
  before insert or update on public.artists
  for each row
  execute function public.set_library_sync_modified();

create trigger albums_sync_modified
  before insert or update on public.albums
  for each row
  execute function public.set_library_sync_modified();


alter table public.artists enable row level security;
alter table public.albums enable row level security;

revoke all on public.artists, public.albums
  from public, anon, authenticated;

grant select, insert, update on public.artists, public.albums
  to authenticated;

create policy artists_own
  on public.artists
  to authenticated
  using ((select auth.uid()) = user_id)
  with check ((select auth.uid()) = user_id);

create policy albums_own
  on public.albums
  to authenticated
  using ((select auth.uid()) = user_id)
  with check ((select auth.uid()) = user_id);


alter publication supabase_realtime
  add table public.artists, public.albums;

notify pgrst, 'reload schema';

commit;
