//go:build integration

package squirrel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func aReply(t *testing.T, store *Store, personID int64, said string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, store.pool.QueryRow(context.Background(), `
		insert into coach_answers (person_id, kind, model, prompt, reply, used)
		values ($1, 'sheet', 'test', 'what now', $2, true) returning id`,
		personID, said).Scan(&id))
	return id
}

func aPersonFor(t *testing.T, store *Store, handle string) int64 {
	t.Helper()
	var id int64
	require.NoError(t, store.pool.QueryRow(context.Background(), `
		insert into people (handle) values ($1)
		on conflict (handle) do update set handle = excluded.handle
		returning id`, handle).Scan(&id))
	_, err := store.pool.Exec(context.Background(),
		`delete from coach_answers where person_id = $1`, id)
	require.NoError(t, err)
	return id
}

// Principle 5 was opened so the coach could be useful, and the cost was stated
// at the time: it can now say something that lands badly on a bad day.
// `coach_answers` has kept every exchange since, for exactly that reason — and
// nothing has ever read one back.
func TestAReplyCanBeSaidToHaveLandedBadly(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)
	p := aPersonFor(t, store, "landed-badly")

	id := aReply(t, store, p, "you have done this three times this week")

	marked, err := store.LandedBadly(ctx, p, id, time.Now())
	require.NoError(t, err)
	require.True(t, marked)

	said, err := store.BadlyLanded(ctx, p, 5)
	require.NoError(t, err)
	require.Equal(t, []string{"you have done this three times this week"}, said)
}

// Pressing it twice is pressing it. The moment this exists to serve is the one
// where there is least to spend on it, and a second press turning into an
// error to read is the opposite of the point.
func TestSayingItTwiceIsSayingItOnce(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)
	p := aPersonFor(t, store, "twice-badly")
	id := aReply(t, store, p, "that is not much for a Tuesday")

	first, err := store.LandedBadly(ctx, p, id, time.Now())
	require.NoError(t, err)
	require.True(t, first)

	again, err := store.LandedBadly(ctx, p, id, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.False(t, again, "a second press reported something to deal with")

	said, err := store.BadlyLanded(ctx, p, 5)
	require.NoError(t, err)
	require.Len(t, said, 1, "one reply, said about twice, became two")
}

// Newest first, and only a handful. The coach is shown what does not land
// here; it is never told how often, which would be a fact about the person
// rather than about the words.
func TestOnlyTheLastFewComeBack(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)
	p := aPersonFor(t, store, "a-few-badly")

	for i, said := range []string{"oldest", "middle", "newest"} {
		id := aReply(t, store, p, said)
		_, err := store.LandedBadly(ctx, p, id, time.Now().Add(time.Duration(i)*time.Minute))
		require.NoError(t, err)
	}

	said, err := store.BadlyLanded(ctx, p, 2)
	require.NoError(t, err)
	require.Equal(t, []string{"newest", "middle"}, said)
}

// And nobody else's replies, ever.
func TestWhatLandedBadlyIsOnlyYours(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)
	mine := aPersonFor(t, store, "mine-badly")
	theirs := aPersonFor(t, store, "theirs-badly")

	id := aReply(t, store, theirs, "somebody else's bad night")
	marked, err := store.LandedBadly(ctx, mine, id, time.Now())
	require.NoError(t, err)
	require.False(t, marked, "one person marked another's reply")

	said, err := store.BadlyLanded(ctx, mine, 5)
	require.NoError(t, err)
	require.Empty(t, said)
}

// Only a reply somebody actually read.
//
// The guard throws answers away, and a rejected one is recorded with
// used = false because it was still paid for. Saying "that landed badly" about
// one of those would be saying it about something nobody ever saw — and worse,
// it would put a sentence the guard already rejected into the next prompt as
// an example of what not to do, which is the one place it could do harm.
func TestOnlyAReplyThatWasSeenCanLandBadly(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)
	p := aPersonFor(t, store, "unseen-badly")

	var rejected int64
	require.NoError(t, store.pool.QueryRow(ctx, `
		insert into coach_answers (person_id, kind, model, prompt, reply, used)
		values ($1, 'sheet', 'test', 'what now', 'a reply the guard threw away', false)
		returning id`, p).Scan(&rejected))

	marked, err := store.LandedBadlyLatest(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, marked, "a reply nobody saw was marked as having landed badly")

	said, err := store.BadlyLanded(ctx, p, 5)
	require.NoError(t, err)
	require.Empty(t, said)
	require.NotZero(t, rejected)
}
