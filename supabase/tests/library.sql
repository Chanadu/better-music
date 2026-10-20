insert into auth.users
values
  ('00000000-0000-0000-0000-000000000001'),
  ('00000000-0000-0000-0000-000000000002');

insert into public.artists(id, user_id, name)
values
  ('other-artist', '00000000-0000-0000-0000-000000000002', 'Other artist');

insert into public.albums(id, user_id, artist_id, title)
values
  ('other-album', '00000000-0000-0000-0000-000000000002', 'other-artist', 'Other album');

select set_config(
  'request.jwt.claim.sub',
  '00000000-0000-0000-0000-000000000001',
  false
);

set role authenticated;

insert into public.artists(id, user_id, name, _modified)
values
  ('own-artist', auth.uid(), 'Own artist', '2000-01-01');

insert into public.albums(id, user_id, artist_id, title)
values
  ('own-album', auth.uid(), 'own-artist', 'Own album');

do $$
declare
  changed integer;

begin
  assert (select count(*) = 1 from public.artists), 'RLS leaked other artist';

  assert (select count(*) = 1 from public.albums), 'RLS leaked other album';

  assert (
    select _modified > '2020-01-01'
    from public.artists
    where id = 'own-artist'
  ), 'client controlled server cursor';

  update public.artists
    set name = 'stolen'
    where id = 'other-artist';
  get diagnostics changed = row_count;

  assert changed = 0, 'cross-user update succeeded';

  update public.albums
    set title = 'stolen'
    where id = 'other-album';
  get diagnostics changed = row_count;

  assert changed = 0, 'cross-user album update succeeded';

  begin
    insert into public.albums(id, user_id, artist_id, title)
    values
      ('forged-album', '00000000-0000-0000-0000-000000000002', 'other-artist', 'Forged');
    raise exception 'cross-user album insert succeeded';
  exception
    when insufficient_privilege then
      null;
  end;

  begin
    insert into public.artists(id, user_id, name)
    values
      ('forged', '00000000-0000-0000-0000-000000000002', 'Forged');
    raise exception 'cross-user insert succeeded';
  exception
    when insufficient_privilege then
      null;
  end;

  begin
    insert into public.albums(id, user_id, artist_id, title)
    values
      ('bad-parent', auth.uid(), 'other-artist', 'Invalid');
    raise exception 'cross-user parent succeeded';
  exception
    when foreign_key_violation then
      null;
  end;

  begin
    insert into public.artists(id, user_id, name)
    values
      ('duplicate', auth.uid(), 'OWN ARTIST');
    raise exception 'case-insensitive duplicate succeeded';
  exception
    when unique_violation then
      null;
  end;

  begin
    update public.albums
      set listened = true, rating = 11
      where id = 'own-album';
    raise exception 'invalid rating succeeded';
  exception
    when check_violation then
      null;
  end;

  begin
    update public.albums
      set rating = 5
      where id = 'own-album';
    raise exception 'queued album rating succeeded';
  exception
    when check_violation then
      null;
  end;

  begin
    update public.albums
      set listened = true, rating = 0
      where id = 'own-album';
    raise exception 'zero rating succeeded';
  exception
    when check_violation then
      null;
  end;

  update public.albums
    set listened = true, rating = 1
    where id = 'own-album';

  assert (
    select rating = 1 from public.albums where id = 'own-album'
  ), 'minimum rating was not accepted';

  update public.albums
    set rating = 10
    where id = 'own-album';

  begin
    update public.albums
      set rating = -1
      where id = 'own-album';
    raise exception 'negative rating succeeded';
  exception
    when check_violation then
      null;
  end;

  begin
    delete from public.albums
      where id = 'own-album';
    raise exception 'hard deletion succeeded';
  exception
    when insufficient_privilege then
      null;
  end;

  update public.albums
    set
      listened = true,
      rating = 8,
      last_edit_ms = 123,
      edit_id = 'edit'
    where id = 'own-album';

  update public.albums
    set _deleted = true
    where id = 'own-album';

  assert (
    select _deleted and rating = 8
    from public.albums
    where id = 'own-album'
  ), 'tombstone hidden';

  insert into public.albums(id, user_id, artist_id, title)
  values
    ('replacement', auth.uid(), 'own-artist', 'OWN ALBUM');

end
$$;

reset role;

set role anon;

do $$
begin
  begin
    perform * from public.artists;
    raise exception 'anonymous read succeeded';
  exception
    when insufficient_privilege then
      null;
  end;

end
$$;

reset role;

select 'Library schema, ownership, constraints, and tombstone checks passed' as result;
