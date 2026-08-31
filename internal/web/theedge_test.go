package web

import (
	"net/http"
	"net/http/httptest"

	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func twoThatComeBack() *fakeStore {
	return &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bins out", Every: 7 * 24 * time.Hour, EveryDays: 7, SinceDays: 8, Active: true},
		{ID: 2, Name: "water the plants", Every: 3 * 24 * time.Hour, EveryDays: 3, SinceDays: 4, Active: true},
	}}
}

// The edge is asked for on its own after a press, because a press answers into
// the conversation and leaves the list above it a moment out of date.
func TestTheEdgeCanBeAskedForOnItsOwn(t *testing.T) {
	f := twoThatComeBack()
	res := askedForTheEdge(t, f, "chores")

	body := res.Body.String()
	require.Equal(t, 200, res.Code)
	require.Contains(t, body, "bins out")
	require.NotContains(t, body, "<!doctype html>", "the edge came back with a page around it")
	require.NotContains(t, body, `class="rail"`, "the edge came back with the rail around it")
}

// And what it answers is current, not what the room drew when you arrived.
func TestTheEdgeIsCurrentAfterTheStoreChanges(t *testing.T) {
	f := twoThatComeBack()
	first := routed(t, f).call(t, "GET", "/r/chores", nil).Body.String()
	require.Contains(t, first, "water the plants")

	f.chores = f.chores[:1]

	again := askedForTheEdge(t, f, "chores").Body.String()

	require.Contains(t, again, "bins out")
	require.NotContains(t, again, "water the plants", "the edge answered with a list that has gone")
}

// Every form the edge draws says which room it is in.
//
// turnView.Room comes off a stored turn and an edge turn is never stored, so
// without saying so every press from the list would land in everything — the
// defect #221 was about, arriving by a new road.
func TestEveryFormTheEdgeDrawsCarriesItsRoom(t *testing.T) {
	f := twoThatComeBack()
	body := routed(t, f).call(t, "GET", "/r/chores", nil).Body.String()
	edge := body[strings.Index(body, `id="edge"`):]

	forms := strings.Count(edge, "<form")
	require.Positive(t, forms, "the edge drew no forms, so this measured nothing")
	require.Equal(t, forms, strings.Count(edge, `name="room" value="chores"`),
		"a form on the chores' list does not say which room it is in")
}

// askedForTheEdge is the request the script makes the moment a press lands.
func askedForTheEdge(t *testing.T, f *fakeStore, room string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", "/r/"+room, nil)
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("X-Edge", "1")
	w := httptest.NewRecorder()
	routed(t, f).mux.ServeHTTP(w, r)
	return w
}
