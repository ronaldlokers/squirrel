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

// A fake house. It records what it was asked for; nothing is asserted inside
// the handler.
//
// `require` in a handler goroutine calls FailNow off the test goroutine, which
// testify documents as undefined and which here reported "it asked for the
// wrong model" about a request the slow case abandons on purpose. What the
// model was is a fact for the test body to check, where a failure belongs.
type fakeHouse struct {
	*House
	asked chan string
}

func housed(t *testing.T, answer string, code int, wait time.Duration) *fakeHouse {
	t.Helper()
	asked := make(chan string, 8)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if err := json.NewDecoder(r.Body).Decode(&in); err == nil {
			if model, ok := in["model"].(string); ok {
				select {
				case asked <- model:
				default:
				}
			}
		}
		time.Sleep(wait)
		if code != http.StatusOK {
			w.WriteHeader(code)
			return
		}
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
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_, _ = w.Write(out)
	}))
	t.Cleanup(srv.Close)
	return &fakeHouse{House: NewHouse(srv.URL, "a small one"), asked: asked}
}

func TestTheHouseAnswersTheOneQuestionItIsAsked(t *testing.T) {
	h := housed(t, "QUESTION", 200, 0)
	said, answered := h.AskedAQuestion(context.Background(), "what now?")
	require.True(t, answered)
	require.True(t, said)
	// Asserted here rather than inside the handler, where a failure would be
	// raised off the test goroutine.
	require.Equal(t, "a small one", <-h.asked, "it asked for the wrong model")

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
