package web

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func post(t *testing.T, m *testMux, path string, form url.Values) *httptest.ResponseRecorder {
	t.Helper()
	return m.call(t, "POST", path, strings.NewReader(form.Encode()))
}

func TestActMovesTheNoteAndComesBack(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	w := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"done"}})

	require.Equal(t, 303, w.Code, "a write answers with See Other so a reload does not repeat it")
	require.Equal(t, "/", w.Header().Get("Location"), "there is one place to come back to")
	require.Equal(t, squirrel.ItemDone, f.items[0].State)
}

func TestEveryTransitionReverses(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"drop"}})
	require.Equal(t, squirrel.ItemDropped, f.items[0].State)

	post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"open"}})
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "back to the pile is a transition like any other")
}

func TestRepeatingATransitionIsANoOpNotAnError(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	first := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"keep"}})
	second := post(t, m, "/pile/act", url.Values{"id": {"1"}, "act": {"keep"}})

	require.Equal(t, 303, first.Code)
	require.Equal(t, 303, second.Code, "a retry is a state assertion, not a failure")
	require.Equal(t, squirrel.ItemKept, f.items[0].State)
}

func TestActRefusesAnUnknownAction(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/act", url.Values{"id": {"1"}, "act": {"delete"}})

	require.Equal(t, 400, w.Code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State)
}

func TestChorePromotesAndTheNoteBecomesDone(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/chore", url.Values{"id": {"1"}, "every": {"every 2 weeks"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, squirrel.ItemDone, f.items[0].State, "there is no chore state")
}

func TestChoreRefusesAnIntervalItCannotRead(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	w := post(t, mounted(t, f), "/pile/chore", url.Values{"id": {"1"}, "every": {"every"}})

	require.Equal(t, 400, w.Code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State, "an unreadable interval must not half-promote")
}
