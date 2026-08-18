-- Search reads every row.
--
-- `strpos(lower(raw_text), lower($2)) > 0` cannot use an index: the function
-- hides the column from the planner, so every search is a sequential scan of
-- every message ever received — commands, taps and notes alike. At today's
-- size that is microseconds and nobody would notice. It grows with the number
-- of thoughts you have ever had, which is the one number in this product that
-- only goes up.
--
-- A trigram index answers a substring match directly. It needs the query to be
-- a LIKE rather than a strpos, which is why SearchItems now escapes the term
-- and uses one — see the comment there for what that escaping is protecting.
--
-- The extension is created if it can be. On a cluster where the application
-- role may not, this migration fails and the boot loop retries forever, which
-- would be worse than the scan it replaces — so it is guarded, and the index is
-- only built when the extension is actually there.
do $$
begin
    create extension if not exists pg_trgm;
exception
    when insufficient_privilege then
        raise notice 'pg_trgm not available to this role; search stays a scan';
end;
$$;

do $$
begin
    if exists (select 1 from pg_extension where extname = 'pg_trgm') then
        create index if not exists items_raw_text_trgm
            on items using gin (lower(raw_text) gin_trgm_ops);
    end if;
end;
$$;
