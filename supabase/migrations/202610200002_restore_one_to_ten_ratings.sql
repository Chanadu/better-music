begin;

update public.albums
  set rating = 1
  where rating = 0;

alter table public.albums
  drop constraint albums_rating_check;

alter table public.albums
  add constraint albums_rating_check
  check (rating between 1 and 10);

commit;
