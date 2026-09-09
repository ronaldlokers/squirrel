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
		require.Contains(t, body, "comes back")
		require.Contains(t, body, `name="every" value="7"`)
		require.Contains(t, body, "2 weeks")
	}
	require.Contains(t, full, "bins out")
}
