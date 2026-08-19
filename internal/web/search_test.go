package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// One field in the lid, one search, both kinds of thing. Typing a chore's name
// used to answer with notes about it and no chore — honest, because the field
// says it searches notes, and surprising anyway, because a person searching
// for a word has not first decided what kind of thing the word belongs to.
func TestSearchAnswersWithChoresAsWellAsNotes(t *testing.T) {
	f := &fakeStore{
		items: []squirrel.Item{note(1, "the bins were not collected", squirrel.ItemOpen)},
		chores: []squirrel.Chore{
			{ID: 1, PersonID: 1, Name: "bins out", Active: true, EverDone: true,
				Every: 14 * 24 * time.Hour, EveryDays: 14, SinceDays: 3},
			{ID: 2, PersonID: 1, Name: "water the ferns", Active: true,
				Every: 7 * 24 * time.Hour, EveryDays: 7},
		},
	}
	body := mounted(t, f).call(t, "GET", "/pile?q=bins", nil).Body.String()

	require.Contains(t, body, "bins out", "the chore itself")
	require.Contains(t, body, "the bins were not collected", "and the notes, as before")
	require.NotContains(t, body, "water the ferns", "a chore that does not say it")
}

// A chore found by searching is a way to reach it, not a second place to act
// on it. Two surfaces that can both complete a chore are two views that can
// disagree about whether it was done.
func TestAChoreFoundBySearchingIsALinkAndNotAControl(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14},
	}}
	body := mounted(t, f).call(t, "GET", "/pile?q=bins", nil).Body.String()

	require.Contains(t, body, `<a class="choreHit" href="/chores">`)
	require.NotContains(t, body, `value="did"`)
	require.NotContains(t, body, `value="retire"`)
}

// A word that only a chore says still found nothing before this, and said so.
func TestSearchDoesNotSayNothingWhenAChoreSaysIt(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		{ID: 1, PersonID: 1, Name: "descale the kettle", Active: true, Every: 30 * 24 * time.Hour, EveryDays: 30},
	}}
	body := mounted(t, f).call(t, "GET", "/pile?q=kettle", nil).Body.String()

	require.Contains(t, body, "descale the kettle")
	require.NotContains(t, body, "nothing says")
}
