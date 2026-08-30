-- What the identity provider says about a person that is not their identity.
--
-- `handle` is not this. It is `preferred_username`, it is unique, and
-- PersonForLogin resolves against it — it does identity work whatever the
-- comment beside Person.Handle claims. A display name may repeat, may be
-- absent, and may change on any login, so it gets its own column and no
-- constraint.
--
-- The picture is bytes rather than the URL Authentik hands over. A remote URL
-- would be a third-party request on every render: it leaks a referrer to the
-- provider, and the service worker caches nothing outside /static/, so the one
-- face on the screen would be the thing that vanishes when the network does —
-- on the product whose offline path was deliberately proved. Fetched once at
-- login, served from this origin, exactly as a note's photograph already is.
--
-- All three are nullable and stay nullable: Authentik need not expose either
-- claim, and everyone who signed in before this migration has neither.
alter table people add column if not exists display_name text;
alter table people add column if not exists face         bytea;
alter table people add column if not exists face_type    text;
