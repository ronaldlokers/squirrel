//go:build integration

package squirrel_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// `!now` with a model behind it. What it chooses is rendered by the same
// message with the same buttons, and the recorded line points at what is on
// screen — nothing downstream knows which of the two produced the offer.

type decideRecord struct {
	shown []string
	kind  string
	refID int64
	text  string
	why   string
}

func deciding(t *testing.T, store *squirrel.Store, personID int64, text string, d *decideRecord) string {
	t.Helper()
	chat, got := chatRecorder(strconv.FormatInt(replyIDs.Add(1), 10))
	a := squirrel.NewApplier(store, nil, chat, nil)
	if d != nil {
		a.SetDecider(func(_ context.Context, _ int64, pickedKind string, _ int64) (
			string, int64, string, string, bool) {
			d.shown = append(d.shown, pickedKind)
			if d.text == "" {
				return "", 0, "", "", false
			}
			return d.kind, d.refID, d.text, d.why, true
		})
	}
	require.NoError(t, a.Apply(context.Background(), itemOf(text), &personID))
	require.Len(t, *got, 1)
	return (*got)[0].message.Text
}

func TestNowRendersWhatTheModelChose(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	other := taskOf(t, store, p, "book the MOT")

	d := &decideRecord{kind: "task", refID: other, text: "book the MOT", why: "it expires on friday"}
	reply := deciding(t, store, p, "!now", d)

	require.Contains(t, reply, "book the MOT")
	require.Contains(t, reply, "it expires on friday")
	require.Equal(t, []string{"task"}, d.shown)
}

// The quiet case, and the one that happens most: the picker's answer stands.
func TestNowKeepsThePickersAnswerWhenTheModelDeclines(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	require.Contains(t, deciding(t, store, p, "!now", &decideRecord{}), "ring the vet")
}

// Nothing to hand over is a normal state, and the model is not invited to find
// something — that would be answering a different question.
func TestNowDoesNotAskTheModelWhenThereIsNothing(t *testing.T) {
	store := withStore(t)
	p := owner(t, store)

	d := &decideRecord{kind: "task", refID: 1, text: "invented", why: "invented"}
	reply := deciding(t, store, p, "!now", d)

	require.Empty(t, d.shown)
	require.NotContains(t, reply, "invented")
}

// The tap that resolves a button has to point at what is on screen. A chosen
// offer rendered with the picker's id would mark the wrong thing done, which
// is the one way this could do real damage.
func TestTheRecordedLinePointsAtWhatWasChosen(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)

	taskOf(t, store, p, "ring the vet")
	other := taskOf(t, store, p, "book the MOT")

	deciding(t, store, p, "!now",
		&decideRecord{kind: "task", refID: other, text: "book the MOT", why: "it expires on friday"})

	line, found, err := store.LineAtPosition(ctx, p, 1)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, line.Item)
	require.Equal(t, other, line.Item.ID)
}
