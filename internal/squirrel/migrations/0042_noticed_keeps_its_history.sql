-- The rack still shows one line per thing. What changes is that the ones before
-- it are no longer thrown away.
--
-- 0037 upserted on (person_id, kind, ref_id) so a second observation replaced
-- the first, because "two lines under one strip is a conversation and this is
-- deliberately not one". That rule is about the rack and it still holds: the
-- read below returns the newest line per thing and the rack draws exactly one.
--
-- What it cost was the answer to "what did it tell me about the boiler last
-- week", which nothing could answer once the row was gone. The strip you have
-- opened is not a rack — it is one thing, looked at on purpose — so the older
-- lines belong there.
drop index if exists noticed_one_per_thing;

create index if not exists noticed_about_one_thing
    on noticed (person_id, kind, ref_id, made_at desc);
