//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The weekly read of the record.

// aLearner records what it was shown and answers with what it is told to.
type aLearner struct {
	shown [][]string
	says  []string
	err   error
}

func (l *aLearner) learn(_ context.Context, _ int64, record []string) ([]string, error) {
	l.shown = append(l.shown, record)
	return l.says, l.err
}

func learning(t *testing.T, store *squirrel.Store, p int64, l *aLearner) *squirrel.Scheduler {
	t.Helper()
	return squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: time.UTC,
		Send:    func(context.Context, string, string) error { return nil },
		OnError: func(error) {},
		Learn:   l.learn,
	})
}

func said(t *testing.T, store *squirrel.Store, p int64, who squirrel.Speaker, words string) {
	t.Helper()
	_, err := store.AppendTurn(context.Background(), p, "buddy", squirrel.Turn{Who: who, Words: words})
	require.NoError(t, err)
}

func TestTheWeeklyPassReadsTheRecordAndKeepsWhatItSays(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said(t, store, p, squirrel.SpeakerYou, "ring the vet")
	said(t, store, p, squirrel.SpeakerBuddy, "Good.")

	l := &aLearner{says: []string{"Phone calls get done."}}
	require.NoError(t, learning(t, store, p, l).KnowingTick(ctx, time.Now()))

	require.Len(t, l.shown, 1, "it did not read the record")
	require.Contains(t, strings.Join(l.shown[0], "\n"), "Them: ring the vet")
	require.Contains(t, strings.Join(l.shown[0], "\n"), "Buddy: Good.")

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Equal(t, []string{"Phone calls get done."}, known)
}

// Once a week and not once a minute. The scheduler ticks every minute, so
// without this the record would be read back fourteen hundred times a day.
func TestTheWeeklyPassRunsOnceAWeek(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said(t, store, p, squirrel.SpeakerYou, "ring the vet")

	l := &aLearner{says: []string{"Phone calls get done."}}
	s := learning(t, store, p, l)
	now := time.Now()

	require.NoError(t, s.KnowingTick(ctx, now))
	require.NoError(t, s.KnowingTick(ctx, now.Add(time.Minute)))
	require.NoError(t, s.KnowingTick(ctx, now.Add(6*24*time.Hour)))
	require.Len(t, l.shown, 1, "it read the record again inside the week")

	require.NoError(t, s.KnowingTick(ctx, now.Add(8*24*time.Hour)))
	require.Len(t, l.shown, 2, "it never read the record again")
}

// Nothing said yet is not an empty conclusion. There is no record to read, so
// nothing is written and the next tick tries again rather than waiting a week
// from a pass that read nothing.
func TestWithNothingSaidNothingIsLearnedAndNothingIsMarked(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	l := &aLearner{says: []string{"should not be called"}}
	require.NoError(t, learning(t, store, p, l).KnowingTick(ctx, time.Now()))

	require.Empty(t, l.shown, "it asked about an empty record")
	at, err := store.LearnedAt(ctx, p)
	require.NoError(t, err)
	require.True(t, at.Year() < 2000, "it marked a pass that never ran")
}

// A model that was unreachable costs nothing and is retried on the next tick.
func TestAnUnreachableModelIsTriedAgain(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said(t, store, p, squirrel.SpeakerYou, "ring the vet")

	l := &aLearner{err: errors.New("no coach")}
	s := learning(t, store, p, l)
	now := time.Now()

	require.NoError(t, s.KnowingTick(ctx, now))
	require.NoError(t, s.KnowingTick(ctx, now.Add(time.Minute)))

	require.Len(t, l.shown, 2, "a failed pass waited a week to try again")
	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Empty(t, known)
}

// With no coach nothing is asked and nothing is written, which is the state
// the product was in for a month and works.
func TestWithNoCoachNothingIsLearned(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said(t, store, p, squirrel.SpeakerYou, "ring the vet")

	s := squirrel.NewScheduler(squirrel.SchedulerOptions{
		Store: store, PersonID: p, ConversationID: "9",
		At: 8 * time.Hour, Location: time.UTC,
		Send:    func(context.Context, string, string) error { return nil },
		OnError: func(error) {},
	})
	require.NoError(t, s.KnowingTick(ctx, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Empty(t, known)
}

// The record is who said it and what was said, and nothing else. What is being
// looked for is how somebody works, and a serialised button is noise a model
// will dutifully find a pattern in.
func TestTheRecordIsWordsAndNotButtons(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	_, err := store.AppendTurn(ctx, p, "buddy", squirrel.Turn{
		Who: squirrel.SpeakerBuddy, Words: "This one.",
		Shown: []byte(`{"cards":[{"title":"the boiler","acts":[{"label":"DONE","action":"/pile/act"}]}]}`),
	})
	require.NoError(t, err)

	l := &aLearner{says: []string{"Phone calls get done."}}
	require.NoError(t, learning(t, store, p, l).KnowingTick(ctx, time.Now()))

	record := strings.Join(l.shown[0], "\n")
	require.Contains(t, record, "Buddy: This one.")
	require.NotContains(t, record, "/pile/act")
	require.NotContains(t, record, "DONE")
}

// A pass that concluded nothing clears what was there — the record is replaced
// rather than accumulated, and keeping last week's would make it older than it
// claims.
func TestAPassThatConcludedNothingClearsWhatWasThere(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	said(t, store, p, squirrel.SpeakerYou, "ring the vet")
	require.NoError(t, store.ReplaceKnowing(ctx, p, []string{"Forms get put off."}, time.Now().Add(-8*24*time.Hour)))

	l := &aLearner{}
	require.NoError(t, learning(t, store, p, l).KnowingTick(ctx, time.Now()))

	known, err := store.Knowing(ctx, p)
	require.NoError(t, err)
	require.Empty(t, known, "last week's survived a pass that concluded nothing")
}
