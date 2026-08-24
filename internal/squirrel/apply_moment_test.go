//go:build integration

package squirrel_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// A time the world imposed, end to end: kept, raised at the moment it matters,
// and closed by one word.

func TestAFixedPointIsKeptFromChat(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	reply := triage(t, store, p, "!at 14:30 dentist, 20 minutes away")
	require.Contains(t, reply, "dentist")
	require.Contains(t, reply, "14:10", "the leaving time is the part nobody works out")

	m, found, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, "dentist", m.Label)
}

// Typed without a command, because the marks of a deliberate fixed point are
// already in the sentence.
func TestAFixedPointIsKeptFromAPlainMessage(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "at 09:00 tomorrow's school run")

	_, found, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)
}

// And the note that merely mentions a time is still a note.
func TestAThoughtWithATimeInItIsStillANote(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	insertItem(t, store, p, "ring the garage at 14:30 about the noise")

	_, found, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found)

	pile, _, err := store.OpenItems(ctx, p, 10)
	require.NoError(t, err)
	require.Len(t, pile, 1)
}

func TestWhatToTakeAttachesToTheNextOne(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!at 14:30 dentist")
	reply := triage(t, store, p, "!bring keys, wallet")
	require.Contains(t, reply, "keys, wallet")

	m, _, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.Equal(t, "keys, wallet", m.Bring)
}

// It outranks everything, including a running timer, and the capacity gate
// never touches it: a low day is the day you most need telling to leave.
func TestAFixedPointOutranksEverythingInsideItsWindow(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", time.Now()))
	_, err := store.StartTimer(ctx, p, "the kitchen", 30*time.Minute, time.Now())
	require.NoError(t, err)

	m, ok := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", time.Now())
	require.True(t, ok)
	kept, err := store.CreateMoment(ctx, p, m)
	require.NoError(t, err)

	o, found, err := store.PickNow(ctx, p, kept.WarnAt().Add(time.Minute), false)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, squirrel.OfferMoment, o.Kind)
	require.Equal(t, "dentist", o.Text)
}

// Outside its window it is nobody's business, which is what makes it safe to
// hold one at all.
func TestAFixedPointIsInvisibleOutsideItsWindow(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, _ := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", time.Now())
	kept, err := store.CreateMoment(ctx, p, m)
	require.NoError(t, err)

	_, found, err := store.PickNow(ctx, p, kept.WarnAt().Add(-time.Hour), false)
	require.NoError(t, err)
	require.False(t, found, "hours before, there is nothing to say")
}

func TestTheWarningIsSaidOnceAndThenNotAgain(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, _ := squirrel.ParseMoment("at 14:30 dentist, 20 minutes away", time.Now())
	kept, err := store.CreateMoment(ctx, p, m)
	require.NoError(t, err)

	_, found, err := store.DueMoment(ctx, p, kept.WarnAt().Add(time.Minute))
	require.NoError(t, err)
	require.True(t, found)

	require.NoError(t, store.MarkMomentSaid(ctx, kept.ID, time.Now()))
	_, found, err = store.DueMoment(ctx, p, kept.WarnAt().Add(2*time.Minute))
	require.NoError(t, err)
	require.False(t, found)
}

// One word closes it, and nothing records whether you went or whether it was
// off — that was never this product's business.
func TestLeavingClosesIt(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	triage(t, store, p, "!at 14:30 dentist")
	require.Contains(t, triage(t, store, p, "!leaving"), "Go")

	_, found, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.False(t, found)
}

// Nothing accrues. A moment that has passed is over: there is no list of
// missed ones and nothing to be behind on.
func TestAPassedMomentIsSimplyOver(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	m, _ := squirrel.ParseMoment("at 14:30 dentist", time.Now())
	kept, err := store.CreateMoment(ctx, p, m)
	require.NoError(t, err)

	_, found, err := store.NextMoment(ctx, p, kept.Starts.Add(time.Minute))
	require.NoError(t, err)
	require.False(t, found)

	_, found, err = store.PickNow(ctx, p, kept.Starts.Add(time.Minute), false)
	require.NoError(t, err)
	require.False(t, found)
}

// The whole path, on a process that thinks it is in UTC.
//
// This is the wiring half of issue #148, and it is the half that matters: the
// parser being right proves nothing if the location never reaches it. Reverting
// `applier.In(...)` makes this fail while every parser test stays green, which
// is exactly how the fault survived the first time.
func TestAFixedPointFromChatIsBookedWhereThePersonIs(t *testing.T) {
	t.Setenv("TZ", "UTC")
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	here, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	chat, got := chatRecorder("tz")
	a := squirrel.NewApplier(store, nil, chat, nil)
	a.In(here)
	require.NoError(t, a.Apply(ctx, itemOf("!at 23:30 test"), &p))
	require.Len(t, *got, 1)

	m, found, err := store.NextMoment(ctx, p, time.Now())
	require.NoError(t, err)
	require.True(t, found)

	// 23:30 in Amsterdam is 21:30 UTC. A process clock would have stored 23:30
	// UTC — two hours late in summer, and the confirmation would have read
	// back exactly as typed either way.
	require.Equal(t, 21, m.Starts.UTC().Hour(),
		"booked on the process's clock rather than the person's")
	require.Equal(t, 30, m.Starts.UTC().Minute())
}
