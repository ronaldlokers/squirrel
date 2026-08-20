//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// `!coach` is the ladder's other half: for the times you cannot name what is
// in the way. Everything about it degrades to `!stuck`.

// askRecord is what the seam was handed, so the tests can check that the thing
// on screen went with the question.
type askRecord struct {
	personID            int64
	kind, said, subject string
	reply               string
	err                 error
}

// coached runs one message through an Applier with a stand-in coach. The seam
// is a func of primitives precisely so a test can be one line.
func coached(t *testing.T, store *squirrel.Store, personID int64, text string, ask *askRecord) string {
	t.Helper()
	chat, got := chatRecorder(strconv.FormatInt(replyIDs.Add(1), 10))
	a := squirrel.NewApplier(store, nil, chat, nil)
	if ask != nil {
		a.SetCoach(func(_ context.Context, p int64, kind, said, subject string) (string, error) {
			ask.personID, ask.kind, ask.said, ask.subject = p, kind, said, subject
			return ask.reply, ask.err
		})
	}
	require.NoError(t, a.Apply(context.Background(), itemOf(text), &personID))
	require.Len(t, *got, 1)
	return (*got)[0].message.Text
}

func TestCoachAnswersInTheCoachsWords(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	ask := &askRecord{reply: "Open the envelope. That is the whole of it."}
	reply := coached(t, store, p, "!coach I can't face the tax thing", ask)

	require.Equal(t, "Open the envelope. That is the whole of it.", reply)
	require.Equal(t, p, ask.personID)
	require.Equal(t, "chat", ask.kind)
	require.Equal(t, "I can't face the tax thing", ask.said)
}

// What is on screen goes with the question, named as such, or the model
// answers about the wrong thing.
func TestCoachSendsWhatYouWouldBeHanded(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	ask := &askRecord{reply: "Ring them. It is two minutes."}
	coached(t, store, p, "!coach everything is too much", ask)

	require.Equal(t, "ring the vet", ask.subject)
}

// Someone who has just typed a paragraph about being stuck has already
// overridden the low-day quiet by asking.
func TestCoachStillHasASubjectOnALowDay(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.NoError(t, store.RecordCheckin(ctx, p, squirrel.MoodWiped, "chat", time.Now()))

	ask := &askRecord{reply: "Ring them. Two minutes."}
	coached(t, store, p, "!coach I don't know where to start", ask)
	require.Equal(t, "ring the vet", ask.subject)
}

// The floor. No key, no network, the budget spent, or a reply the guard threw
// away — all four end in the same place, and it is the product answering
// rather than apologising.
func TestCoachFallsBackToTheLadderWhenThereIsNothingToHand(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	ask := &askRecord{err: errors.New("no coach available")}
	require.Contains(t, coached(t, store, p, "!coach help", ask), "!stuck")
}

// With something to hand, the floor is the picker. Someone who has just typed
// out five things wants one of them chosen, and the product can choose one
// without a model — that is what PickNow is for. Answering a pile with a
// question about a thing nobody has named yet would be worse than useless.
func TestCoachThatCannotAnswerHandsYouTheOneThing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	ask := &askRecord{err: errors.New("no coach available")}
	reply := coached(t, store, p, "!coach the tax thing, the vet, the bins and mum", ask)

	require.Contains(t, reply, "ring the vet")
	require.NotContains(t, reply, "!stuck")
}

func TestCoachWithNoCoachWiredSaysSoAndPointsAtTheLadder(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	reply := coached(t, store, p, "!coach help", nil)
	require.Contains(t, reply, "No coach here")
	require.Contains(t, reply, "!stuck")
}

func TestCoachWithNothingSaidAsksForWords(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	ask := &askRecord{reply: "should not be called"}
	reply := coached(t, store, p, "!coach", ask)

	require.Contains(t, reply, "Say what is going on")
	require.Empty(t, ask.said, "nothing is asked of the model")
}

// Help does not advertise a command with nothing behind it.
func TestHelpMentionsTheCoachOnlyWhenThereIsOne(t *testing.T) {
	t.Cleanup(func() { squirrel.SetCoachHere(false) })

	squirrel.SetCoachHere(false)
	require.NotContains(t, squirrel.HelpMessage().Text, "!coach")

	squirrel.SetCoachHere(true)
	help := squirrel.HelpMessage().Text
	require.Contains(t, help, "!coach <words>")
	require.Contains(t, help, "!stuck", "the ladder is still the first thing offered")
}
