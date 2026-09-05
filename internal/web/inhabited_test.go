package web

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestNoControlRenamesItself(t *testing.T) {
	f := &fakeStore{
		items:   []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)},
		checkin: &squirrel.Checkin{Mood: squirrel.MoodCalm, SaidAt: time.Now()},
	}

	deck := opened(t, f, "notes")
	for _, label := range []string{"done", "keep", "drop", "make a chore"} {
		require.Contains(t, deck, label, "the pile stopped saying %q", label)
	}
}
