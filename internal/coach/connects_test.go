package coach_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// What the clause is allowed to point at.

func messagesIn(t *testing.T, sent map[string]any) string {
	t.Helper()
	msgs, ok := sent["messages"].([]any)
	require.True(t, ok)
	var b strings.Builder
	for _, one := range msgs {
		words, _ := one.(map[string]any)["content"].(string)
		b.WriteString(words + "\n")
	}
	return b.String()
}

func TestTheRestOfTheBoardReachesTheModel(t *testing.T) {
	f := &fakeFacts{
		work:    []coach.Work{{ID: 7, Kind: "task", Text: "ring the council about the bins"}},
		written: []coach.Written{{Text: "the number for the council is 0117 922 2100"}},
	}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "the number for it is written down already",
		})),
	)

	d, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.NoError(t, err)
	require.Equal(t, "the number for it is written down already", d.Because)

	said := messagesIn(t, api.sent[0])
	require.Contains(t, said, "the number for the council is 0117 922 2100",
		"the clause has nothing to point at")
	require.Contains(t, said, "You cannot choose any of these")
}

// A note has no id anywhere in what the model was given, so choosing one means
// inventing a number — and an invented number is refused the same as any other.
func TestANoteIsNotSomethingThatCanBeHandedOver(t *testing.T) {
	f := &fakeFacts{
		work:    []coach.Work{{ID: 7, Kind: "task", Text: "ring the council"}},
		written: []coach.Written{{Text: "book the garage for the MOT"}},
	}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 41, "because": "it is written down",
		})),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.ErrorIs(t, err, coach.ErrUnavailable, "a note was handed over as a thing to do")

	require.NotContains(t, messagesIn(t, api.sent[0]), `"id"`,
		"the notes went in carrying something that can be chosen")
}

func TestAnEmptyBoardSaysNothingAboutItself(t *testing.T) {
	f := &fakeFacts{work: []coach.Work{{ID: 7, Kind: "task", Text: "ring the council"}}}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "it is short",
		})),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.NoError(t, err)

	require.NotContains(t, messagesIn(t, api.sent[0]), "Also on the board",
		"an empty board was described as if it had something on it")
}

// The clause is better without a connection than the offer is without a clause.
func TestABoardThatCannotBeReadStillChooses(t *testing.T) {
	f := &fakeFacts{
		work:       []coach.Work{{ID: 7, Kind: "task", Text: "ring the council"}},
		writtenErr: errors.New("no database"),
	}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "it is short",
		})),
	)

	d, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.NoError(t, err, "a board that could not be read took the offer with it")
	require.Equal(t, int64(7), d.RefID)
	require.NotContains(t, messagesIn(t, api.sent[0]), "Also on the board")
}

func TestABlankNoteIsNotShown(t *testing.T) {
	f := &fakeFacts{
		work:    []coach.Work{{ID: 7, Kind: "task", Text: "ring the council"}},
		written: []coach.Written{{Text: "   "}, {Text: "the number is 0117 922 2100"}},
	}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "choose", map[string]any{
			"kind": "task", "ref_id": 7, "because": "it is short",
		})),
	)

	_, err := deciderFor(api, f, &fakeLog{}).Decide(context.Background(), 1)
	require.NoError(t, err)

	said := messagesIn(t, api.sent[0])
	require.NotContains(t, said, "- \n", "a blank line was handed over as a note")
	require.Contains(t, said, "the number is 0117 922 2100")
}
