# When it breaks

What to do at two in the morning, written down so it does not depend on
remembering how any of this works.

This repository owns the code. The cluster manifests, the volumes and the
backups live in [ronaldlokers/homelab](https://github.com/ronaldlokers/homelab),
so the *procedures* here point at that repository for the commands.

---

## A migration that will not apply

**What you will see.** The pod is up and answering. Captures are still
accepted — the spool takes them and nothing about a migration blocks that. But
nothing is draining, no nudge arrives, and the log repeats, once per drain
interval:

```
level=ERROR msg="a migration will not apply; the schema is at the last one that
did and the drain is not running"
```

**What is true.** Each migration applies inside a transaction that also records
it in `schema_migrations`, so a migration either lands completely or not at
all. There is no half-applied schema. What you have is a schema at the last
migration that succeeded, and a binary that expects one further along.

Two consequences worth knowing before you touch anything:

- **Nothing is lost while this is happening.** Captures go to the fsynced
  spool. They drain when the schema catches up. Leaving it broken overnight
  costs you the evening message, not the notes.
- **The previous image still runs.** It expects the schema this database
  actually has.

**What to do.**

1. Read the error. It names the file: `applying migrations/00NN_thing.sql`.
2. Check what the database thinks it has applied:

   ```sql
   select version, applied_at from schema_migrations order by version desc limit 5;
   ```

   The newest row is where you are. The failing file is the one after it.
3. **Roll the image back** to the previous tag in `homelab`. The old binary and
   the current schema agree, the drain resumes, and the pile catches up. Do
   this first — it ends the outage — and fix the migration in daylight.
4. Fix forward. There are no down migrations and there should not be: a
   migration that has never been applied does not need reverting, and one that
   has is history. Edit the failing file if it has never applied anywhere;
   write `00NN+1` if it has applied in one cluster and not the other.

**What not to do.** Do not hand-edit `schema_migrations` to mark the failing
migration as applied. The row means "this ran"; a row that lies means the next
person — you, later — cannot tell what the schema is.

---

## The restore drill

Squirrel's state is in two places that must be restored **together**:

- **Postgres** — every note, chore, timer, check-in and coach exchange.
- **The photographs volume** — the files behind `attachment_path`. Chosen as a
  PVC on the pod rather than object storage, deliberately, and the cost of that
  choice is this section: the drill has to cover the volume, or a photograph is
  a note you have lost.

A note whose only content is a photograph is not decorated by that file; it
**is** that file. Restoring one without the other leaves either rows pointing
at nothing or files nothing points at.

### Doing it

Run against a scratch namespace, not production.

1. **Take the pair.** Dump the database and snapshot the volume as close
   together as you can, and **in that order — the dump first, the volume
   second.** A photograph is written to disk before its row is inserted, so
   whichever half is taken last decides which way a restored pair can be wrong:

   | order | what a restore gives you |
   | --- | --- |
   | volume, then dump | a row pointing at a file no backup holds |
   | dump, then volume | a file nothing points at |

   The second is invisible and harmless. The first is a note rendering a broken
   picture, and it is the direction this drill exists to catch.

   This step said the opposite until 5 September 2026, and gave the right
   reason for it. The nightly jobs followed the same wrong order — the volume
   at 00:00 UTC, the database dump at 03:00 — leaving three hours every night
   in which a photograph had its row in the dump and its file in no backup at
   all. That is fixed in `homelab`: the photographs volume has its own backup
   job that runs an hour after the dump.
2. **Restore both** into the scratch namespace and point a Squirrel at them.
3. **Check the join, which is the whole point of the drill:**

   ```sql
   select count(*) from items where attachment_path is not null;
   ```

   Then open `/pile` and load each of those notes' photographs. A `404` in the
   log — *"a photograph the row expects is not on disk"* — is the drill failing
   and is exactly what it is for.
4. **Check the drain still runs.** The log says `database ready` and the
   evening message renders.
5. **Write down the date it passed** in the table below.

### When it was last proved

| Date | Postgres | Photographs | By |
| --- | --- | --- | --- |
| 2026-09-05 | Restored from `pg_dump` into a scratch namespace. 66 of 66 rows; the `attachment_path` join returns 1, row 38, matching production. | Restored from Longhorn backup `squirrel-drill-backup-20260905`. The file is present, 2,598,750 bytes, `ff d8 ff` at the head and `ff d9` at the tail, so not truncated. The thumb is there too. | Ronald |

That row is not a placeholder to be tidied away. Until it has a date in it,
"capture is sacred" is a promise about photographs that nothing has tested.

**Closing the byte-for-byte gap.** The first run could not compare the
restored file against the live volume, because there was no way to read that
volume without either a shell in a distroless pod or a second pod mounting an
RWO claim the running one holds. `GET /photo/{id}/checksum` closes it: it is
an authenticated route, guarded the same as `/photo/{id}`, that reports a
photograph's size in bytes and its SHA-256 rather than the bytes themselves —
so the running app answers for its own volume over HTTP, and nothing needs a
shell or a second claim.

To use it: for one of the notes step 3 found, hit `/photo/{id}/checksum`
against the **live** Squirrel and again against the **restored** one in the
scratch namespace, and compare the two JSON bodies. A match is the
byte-for-byte proof; anything else is the drill failing, the same as a `404`
in step 3.

**The drill took about eleven minutes**, dump to teardown, and the database is
10 MB. It is a cheap thing to do and there is no reason for the next gap to be
fifteen days.

### Repeat it after

- any migration that touches `items.attachment_path` or `attachment_type`;
- any change to where photographs are written (`PHOTO_DIR`);
- moving or resizing the volume.

---

## The volume filling up

`PHOTO_CEILING_BYTES` is the volume's size in bytes, set in `homelab`. Past
four fifths of it, a successful capture warns once:

```
level=WARN msg="the photographs are filling their volume" used=… ceiling=…
```

Nothing deletes a photograph, and nothing should: it is part of what was
captured. Grow the volume and raise the ceiling to match.

Unset, there is no ceiling and no warning — which is a supported choice and the
default, and means the first thing you hear is a capture failing.

---

## Buddy has gone quiet

Every reason collapses to one behaviour on purpose: the picker chooses, the
ladder answers, and the product works exactly as it did before the model
existed. That is Rule 10 and it is tested. But the reasons are different and
the logs tell them apart:

| Log | What happened | What to do |
| --- | --- | --- |
| `the coach is over its budget for the month; …` | The ceiling did its job. | Nothing. It resets on the first. |
| `the coach … why=refused` | The provider answered and said no: a retired model id, a revoked key, a rate limit. | Check the model ids in `internal/coach/prices.go` against what the provider still serves, then the key. **This is the one that stays broken for weeks if nobody looks** — every call fails identically and the replies just get blander. |
| `the coach … why=unreachable` | The provider did not answer. | Wait. If it persists, check `COACH_BASE_URL` and the network. |
| `the coach … why=nonsense` | Something answered that was not a completion — nearly always a proxy or an auth page in front of the provider. | Check `COACH_BASE_URL` points at the API and not at a login. |
| `the coach said something the wrong shape` | The guard rejected a reply. The fixed answer went instead. | Nothing, unless it is every reply. |
| nothing at all | No key is configured, so no coach was ever built. | Nothing, if that is deliberate. |

`/buddy` redirects now; there is no lid to check spend in. The spend against
the ceiling rides on Buddy's own reply, in the conversation — `costLine` in
`internal/web/coach.go`, rendered as the `.cost` line under that turn — and
nowhere else. It only appears on a reply that actually asked the model
something; nothing shows a running total on its own.
