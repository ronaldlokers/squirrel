-- A third answer to an offer: 'wrong' says the pick itself was a mistake,
-- distinct from 'later' — a deferral, which concedes the pick and expires at
-- midnight. 'wrong' does not expire; the picker's own Suppressed reads it with
-- no time floor.
alter table offers drop constraint offers_answer_check;
alter table offers add constraint offers_answer_check
    check (answer in ('later', 'did', 'started', 'wrong'));
