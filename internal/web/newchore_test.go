package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheChoresRackTeachesHowToMakeOne(t *testing.T) {
	full := theChores(t, &fakeStore{chores: []squirrel.Chore{
		{ID: 1, Name: "bins out", Active: true, Every: 14 * 24 * time.Hour, EveryDays: 14},
	}})
	empty := theChores(t, &fakeStore{})

	for _, body := range []string{full, empty} {
		require.Contains(t, body, "what comes back?")
		require.Contains(t, body, `name="every" type="number"`)
		require.Contains(t, body, `name="unit"`)
		require.Contains(t, body, `<option value="weeks">weeks</option>`)
	}
	require.Contains(t, full, "bins out")
}
