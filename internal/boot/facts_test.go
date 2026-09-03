//go:build integration

package boot

import (
	"context"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// What the model may see. This is the file where the caps and the suppressions
// are actually enforced — everything above it trusts that they were.
//
// In package boot rather than boot_test, because factsOver is unexported and
// the point of these tests is the adapter itself. That means its own small
// helpers rather than testsupport_test.go's, which live in the other package.

func factsStore(t *testing.T) *squirrel.Store {
	t.Helper()
	ctx := context.Background()

	store, err := squirrel.OpenStore(ctx, os.Getenv("TEST_DATABASE_URL"))
	require.NoError(t, err)
	t.Cleanup(store.Close)

	require.NoError(t, store.Migrate(ctx))
	_, err = store.Pool().Exec(ctx,
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
	return store
}

func factsOwner(t *testing.T, store *squirrel.Store) int64 {
	t.Helper()
	id, err := store.SeedOwner(context.Background(), "ronald", nil)
	require.NoError(t, err)
	return id
}

func factsFor(t *testing.T, store *squirrel.Store, at time.Time) *facts {
	t.Helper()
	f := factsOver(store)
	f.now = func() time.Time { return at }
	return f
}

func taskFor(t *testing.T, store *squirrel.Store, personID int64, text string) int64 {
	t.Helper()
	ctx := context.Background()
	id, err := store.InsertItemReturningID(ctx, squirrel.Item{
		PersonID: &personID, RawText: text, ReceivedAt: time.Now(), Transport: "test",
		ExternalID: squirrel.Ptr(text), Payload: []byte(`{}`),
	})
	require.NoError(t, err)
	_, err = store.SetItemKind(ctx, personID, id, squirrel.ItemTask)
	require.NoError(t, err)
	return id
}

// overdueChore is a chore whose interval has already passed, since a chore
// created a moment ago is not due and DueChores is what OpenWork reads.
func overdueChore(t *testing.T, store *squirrel.Store, personID int64, name string) {
	t.Helper()
	ctx := context.Background()

	c, err := store.UpsertChore(ctx, personID, name, 14*24*time.Hour, 30*time.Minute)
	require.NoError(t, err)
	_, err = store.Pool().Exec(ctx,
		`update chores set created_at = now() - make_interval(secs => $2) where id = $1`,
		c.ID, int64((15 * 24 * time.Hour).Seconds()))
	require.NoError(t, err)
}

func TestOpenWorkIsTasksAndDueChoresAndNeverThePile(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	taskFor(t, store, p, "ring the vet")
	overdueChore(t, store, p, "put the bins out")
	// A note is a thought you have not decided about. Handing the pile to
	// something that chooses would make the model do the deciding, which is
	// the one thing triage exists to stop it doing.
	_, err := store.InsertItem(ctx, squirrel.Item{
		PersonID: &p, RawText: "the boiler makes a noise", ReceivedAt: now, Transport: "test",
		ExternalID: squirrel.Ptr("note-1"), Payload: []byte(`{}`),
	})
	require.NoError(t, err)

	work, err := factsFor(t, store, now).OpenWork(ctx, p, 10)
	require.NoError(t, err)

	said := map[string]string{}
	for _, w := range work {
		said[w.Text] = w.Kind
	}
	require.Equal(t, "task", said["ring the vet"])
	require.Equal(t, "chore", said["put the bins out"])
	require.NotContains(t, said, "the boiler makes a noise", "the pile reached the model")
}

// "Not now" has to mean the same thing whichever of the two is choosing. The
// suppression belongs in the fact rather than in a prompt, for the same reason
// the caps do.
func TestOpenWorkHidesWhatWasTurnedDownToday(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	id := taskFor(t, store, p, "ring the vet")
	taskFor(t, store, p, "book the MOT")
	require.NoError(t, store.Refuse(ctx, p, squirrel.OfferTask, id, now))

	work, err := factsFor(t, store, now).OpenWork(ctx, p, 10)
	require.NoError(t, err)

	for _, w := range work {
		require.NotEqual(t, "ring the vet", w.Text, "a refused thing was offered anyway")
	}
	require.NotEmpty(t, work)
}

// A cap the model is asked to respect is a cap it can ignore, so it is applied
// here.
func TestOpenWorkIsCapped(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)

	for _, text := range []string{"one", "two", "three", "four", "five"} {
		taskFor(t, store, p, text)
	}

	work, err := factsFor(t, store, time.Now()).OpenWork(ctx, p, 2)
	require.NoError(t, err)
	require.Len(t, work, 2)
}

// The leave-by arithmetic is done before the model sees it. Handing it the
// parts and letting it subtract would be two answers to one question, and the
// wrong one makes you late.
func TestNextFixedCarriesTheLeaveByAlreadyWorkedOut(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	_, err := store.CreateMoment(ctx, p, squirrel.Moment{
		Label: "dentist", Starts: now.Add(90 * time.Minute), Travel: 20 * time.Minute,
	})
	require.NoError(t, err)

	fixed, found, err := factsFor(t, store, now).NextFixed(ctx, p)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "dentist", fixed.Label)
	require.InDelta(t, 70, fixed.LeaveIn, 1)
}

func TestNextFixedIsAbsentWhenNothingIsComing(t *testing.T) {
	store := factsStore(t)
	p := factsOwner(t, store)

	_, found, err := factsFor(t, store, time.Now()).NextFixed(context.Background(), p)
	require.NoError(t, err)
	require.False(t, found)
}

// Done only, and never a total. The one count the store returns is dropped on
// the floor here rather than repeated to something that would say it out loud.
func TestLatelyIsWhatWasDoneAndCarriesNoNumber(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	id := taskFor(t, store, p, "ring the vet")
	require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, now))

	lately, err := factsFor(t, store, now).Lately(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, lately, 1)
	require.Equal(t, "ring the vet", lately[0].What)
}

func TestLatelyIsCapped(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	for _, text := range []string{"one", "two", "three"} {
		id := taskFor(t, store, p, text)
		require.NoError(t, store.SetItemState(ctx, id, squirrel.ItemDone, now))
	}

	lately, err := factsFor(t, store, now).Lately(ctx, p, 2)
	require.NoError(t, err)
	require.Len(t, lately, 2)
}

// One id, one person. A model asking for an id that is not yours must not get
// an answer, and the store's own scoping is what guarantees it.
func TestItemIsOnlyEverYourOwn(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	mine := factsOwner(t, store)

	id := taskFor(t, store, mine, "ring the vet")
	f := factsFor(t, store, time.Now())

	w, found, err := f.Item(ctx, mine, id)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "ring the vet", w.Text)

	_, found, err = f.Item(ctx, mine+1000, id)
	require.NoError(t, err)
	require.False(t, found)
}

// The clock is a signal, not a diagnosis: "low", never a mood word and never a
// history.
func TestClockCarriesCapacityAndNoMoodWord(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()

	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "test", now))

	clock, err := factsFor(t, store, now).Clock(ctx, p)
	require.NoError(t, err)
	require.Equal(t, "low", clock.Capacity)
	require.NotEmpty(t, clock.Clock)
}

// The sixth read tool. What it can say is how long something took; what it
// cannot say is how often you did not finish, because those runs never exist.
func TestTypicallyReadsFinishedRunsOnly(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	now := time.Now()
	f := factsFor(t, store, now)

	// Stopped early, five times over. Not a measurement.
	for i := 0; i < 5; i++ {
		_, err := store.StartTimer(ctx, p, "the kitchen", 10*time.Minute,
			now.Add(time.Duration(i)*time.Hour))
		require.NoError(t, err)
		require.NoError(t, store.StopTimer(ctx, p))
	}
	_, found, err := f.Typically(ctx, p, "the kitchen")
	require.NoError(t, err)
	require.False(t, found, "stopping early reached the model as a measurement")

	// Ran to the end, three times. That is one.
	for i := 0; i < 3; i++ {
		at := now.Add(time.Duration(10+i) * time.Hour)
		_, err := store.StartTimer(ctx, p, "put the bins out", 10*time.Minute, at)
		require.NoError(t, err)
		_, ok, err := store.ClaimFinishedTimer(ctx, p, at.Add(10*time.Minute))
		require.NoError(t, err)
		require.True(t, ok)
	}

	minutes, found, err := f.Typically(ctx, p, "put the bins out")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, 10, minutes)
}

// noteFor is a thought nobody has decided about, which is what Written reads
// and what OpenWork refuses.
func noteFor(t *testing.T, store *squirrel.Store, personID int64, text string) {
	t.Helper()
	_, err := store.InsertItem(context.Background(), squirrel.Item{
		PersonID: &personID, RawText: text, ReceivedAt: time.Now(), Transport: "test",
		ExternalID: squirrel.Ptr(text), Payload: []byte(`{}`),
	})
	require.NoError(t, err)
}

// The other half of OpenWork's refusal. The pile stays out of what may be
// chosen and comes in here instead, where the clause can point at it.
func TestWrittenIsThePileAndNothingElse(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)

	noteFor(t, store, p, "the number for the council is 0117 922 2100")
	taskFor(t, store, p, "ring the council about the bins")
	overdueChore(t, store, p, "put the bins out")

	written, err := factsFor(t, store, time.Now()).Written(ctx, p, 10)
	require.NoError(t, err)

	said := []string{}
	for _, one := range written {
		said = append(said, one.Text)
	}
	require.Equal(t, []string{"the number for the council is 0117 922 2100"}, said,
		"something that was already decided about came back as a note")
}

func TestWrittenIsCapped(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	p := factsOwner(t, store)
	for i := range 14 {
		noteFor(t, store, p, "a thought number "+strconv.Itoa(i))
	}

	written, err := factsFor(t, store, time.Now()).Written(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, written, 10)
}

func TestWrittenIsOnlyEverYourOwn(t *testing.T) {
	ctx := context.Background()
	store := factsStore(t)
	mine := factsOwner(t, store)
	theirs, err := store.PersonForLogin(ctx, "sub-someone-else", "someone-else")
	require.NoError(t, err)

	noteFor(t, store, theirs, "their thought")
	noteFor(t, store, mine, "my thought")

	written, err := factsFor(t, store, time.Now()).Written(ctx, mine, 10)
	require.NoError(t, err)
	require.Len(t, written, 1, "somebody else's note reached the clause")
	require.Equal(t, "my thought", written[0].Text)
}
