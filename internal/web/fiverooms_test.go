package web

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTheRoomsThatStoppedBeingRoomsStillLandSomewhere(t *testing.T) {
	for from, to := range map[string]string{
		"/r/buddy": "/", "/r/everything": "/", "/r/pile": "/?bay=notes",
		"/r/held": "/?shelf=held", "/r/kept": "/?shelf=kept",
		"/r/notes": "/?bay=notes", "/r/chores": "/?bay=daily", "/r/at": "/", "/r/tasks": "/?bay=once",
	} {
		res := mounted(t, &fakeStore{}).call(t, "GET", from, nil)

		require.Equal(t, 301, res.Code, "%s answers %d", from, res.Code)
		require.Equal(t, to, res.Header().Get("Location"), "%s lands in the wrong place", from)
	}
}
