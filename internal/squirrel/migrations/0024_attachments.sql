-- A photograph of the thing, instead of typing what it says.
--
-- The case is a letter, a serial plate, a parking sign: information you want
-- kept and do not want to transcribe. Camera-first from the screen, because
-- the moment you want this is the moment you are standing in front of the
-- thing holding a phone.
--
-- Columns on items rather than a table of their own. One note has at most one
-- photograph, which is what the case wants and is a bound worth having: a note
-- that can carry five pictures is an album, and an album is a thing you
-- organise rather than a thing you glance at.
alter table items add column if not exists attachment_path text;
alter table items add column if not exists attachment_type text;

-- What makes a row worth showing, decided once.
--
-- Seven queries filtered on `raw_text <> ''` before this. The filter exists
-- because an attachment-only Campfire message lands as a row with no body —
-- the fork sends the filename, not the bytes — and a blank numbered line in
-- the pile is a thought lost silently.
--
-- A row with a photograph in it is not blank, so every one of those seven had
-- to learn the difference. Seven copies of a definition is seven chances to
-- update six of them, and the one that got missed would hide a note rather
-- than show a blank one — the same failure, in the direction nobody notices.
--
-- So the definition lives here, in the column, and the queries ask it. It is
-- stored rather than computed on read so the existing index over state and
-- received_at is still the whole plan for the pile.
alter table items add column if not exists has_content boolean
  generated always as (raw_text <> '' or attachment_path is not null) stored;
