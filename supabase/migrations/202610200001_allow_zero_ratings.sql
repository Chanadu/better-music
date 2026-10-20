begin;

alter table public.albums
  drop constraint albums_rating_check;

alter table public.albums
  add constraint albums_rating_check
  check (rating between 0 and 10);

commit;
