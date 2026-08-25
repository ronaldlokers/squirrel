package coach

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// The model in the house.

func housed(t *testing.T, answer string, code int, wait time.Duration) *House {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(wait)
		if code != http.StatusOK {
			w.WriteHeader(code)
			return
		}
		var in map[string]any
		_ = json.NewDecoder(r.Body).Decode(&in)
		require.Equal(t, "a small one", in["model"], "it asked for the wrong model")
		w.Header().Set("Content-Type", "application/json")
		// Marshalled rather than concatenated: one of the cases is an answer
		// wrapped in quotation marks, and pasting that into a JSON string
		// makes the fixture invalid rather than the case it is testing.
		out, err := json.Marshal(chatResponse{Choices: []struct {
			Message struct {
				Content string     `json:"content"`
				Calls   []toolCall `json:"tool_calls"`
			} `json:"message"`
		}{{Message: struct {
			Content string     `json:"content"`
			Calls   []toolCall `json:"tool_calls"`
		}{Content: answer}}}})
		require.NoError(t, err)
		_, _ = w.Write(out)
	}))
	t.Cleanup(srv.Close)
	return NewHouse(srv.URL, "a small one")
}

func TestTheHouseAnswersTheOneQuestionItIsAsked(t *testing.T) {
	said, answered := housed(t, "QUESTION", 200, 0).AskedAQuestion(context.Background(), "what now?")
	require.True(t, answered)
	require.True(t, said)

	said, answered = housed(t, "THOUGHT", 200, 0).AskedAQuestion(context.Background(), "the boiler")
	require.True(t, answered)
	require.False(t, said)
}

// A small model that answered with punctuation or a capital is still
// answering. This is the one place a tolerant read is right: the alternative
// is falling back to a rule that would have said the same thing.
func TestTheHouseIsReadTolerantly(t *testing.T) {
	for _, answer := range []string{"question", "QUESTION.", " Question ", `"QUESTION"`} {
		said, answered := housed(t, answer, 200, 0).AskedAQuestion(context.Background(), "x")
		require.True(t, answered, "%q was not understood", answer)
		require.True(t, said)
	}
}

// Anything that is not one of the two words is no answer rather than a guess.
// The rule underneath is better than a coin toss.
func TestAnythingElseIsNoAnswer(t *testing.T) {
	for _, answer := range []string{"maybe", "I think that is a question", ""} {
		_, answered := housed(t, answer, 200, 0).AskedAQuestion(context.Background(), "x")
		require.False(t, answered, "%q was treated as an answer", answer)
	}
}

// Asleep, unreachable, or thinking for longer than a press takes.
func TestAHouseThatIsSlowIsAHouseThatDidNotAnswer(t *testing.T) {
	h := housed(t, "QUESTION", 200, houseTimeout+300*time.Millisecond)

	_, answered := h.AskedAQuestion(context.Background(), "what now?")

	require.False(t, answered, "it waited longer than a press takes")
}

func TestAHouseThatRefusesIsAHouseThatDidNotAnswer(t *testing.T) {
	_, answered := housed(t, "", 500, 0).AskedAQuestion(context.Background(), "what now?")
	require.False(t, answered)
}

// No address is no house, rather than a house that cannot work.
func TestNoAddressIsNoHouse(t *testing.T) {
	require.Nil(t, NewHouse("", "a small one"))
	require.Nil(t, NewHouse("http://the-house", ""))
	require.NotNil(t, NewHouse("http://the-house", "a small one"))
}

// And a nil one answers nothing rather than panicking, because the caller's
// own nil check is the configuration and a second one here is cheap.
func TestANilHouseAnswersNothing(t *testing.T) {
	var none *House
	_, answered := none.AskedAQuestion(context.Background(), "what now?")
	require.False(t, answered)
}

// It costs electricity in a cupboard rather than money abroad, so nothing
// about it touches the budget — the House struct has no budget to touch, which
// is the property rather than an intention.
func TestTheHouseIsNotBudgeted(t *testing.T) {
	h := NewHouse("http://the-house", "a small one")

	require.Nil(t, h.Client.Transport, "it was given something to authenticate with")
	require.Equal(t, houseTimeout, h.Client.Timeout,
		"it may wait longer than a press takes")
}
