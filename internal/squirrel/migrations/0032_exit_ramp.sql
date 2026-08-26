-- The hyperfocus exit ramp.
--
-- The thing about hyperfocus is not that you cannot stop. It is that the
-- decision to stop never arrives: there is no moment at which you notice, so
-- there is no moment at which you choose. A timer that has run out and been
-- ignored for half an hour is the only evidence this product has that it is
-- happening.
--
-- **Opt-in, at the moment you start the timer.** An app that decided on its own
-- to interrupt somebody deep in a thing would be the exact opposite of what
-- every other line in this product is for, and the version of this that feels
-- like nagging is the version that gets switched off after two days.
alter table timers add column if not exists ramp boolean not null default false;

-- Set when it has spoken. Null while it has not, and cleared by StartTimer
-- along with the rest — a new timer is a new decision, so a new timer may
-- speak once too.
alter table timers add column if not exists ramp_said_at timestamptz;

-- "leave me alone", and this is the one column StartTimer does not clear.
--
-- ramp_said_at already stops it speaking twice about one timer. This stops it
-- speaking about the *next* one, which is what somebody means when they say
-- leave me alone — they are not talking about this timer, they are talking
-- about today.
alter table timers add column if not exists ramp_hushed_until timestamptz;
