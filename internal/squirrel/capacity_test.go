package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// No build tag: this is the whole of "a low day" and it touches no database,
// so it should fail on a laptop with no Postgres rather than only in CI.

func TestCapacityIsLowOnlyForTheTwoCapacityWords(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)

	for _, tc := range []struct {
		mood squirrel.Mood
		want squirrel.Capacity
	}{
		{squirrel.MoodGood, squirrel.CapacityOK},
		{squirrel.MoodCalm, squirrel.CapacityOK},
		// Flat but functional. An empty day here would read as the product
		// agreeing you are finished, which is the opposite of the help wanted.
		{squirrel.MoodLow, squirrel.CapacityOK},
		{squirrel.MoodFrazzled, squirrel.CapacityLow},
		{squirrel.MoodWiped, squirrel.CapacityLow},
	} {
		c := squirrel.Checkin{Mood: tc.mood, SaidAt: now.Add(-time.Minute)}
		require.Equal(t, tc.want, squirrel.CapacityOf(c, true, now), string(tc.mood))
	}
}

func TestCapacityIgnoresAStaleReading(t *testing.T) {
	now := time.Date(2026, 8, 20, 14, 0, 0, 0, time.UTC)
	// Seven hours ago is this morning, and this morning is not now.
	stale := squirrel.Checkin{Mood: squirrel.MoodWiped, SaidAt: now.Add(-7 * time.Hour)}

	require.Equal(t, squirrel.CapacityOK, squirrel.CapacityOf(stale, true, now),
		"a rough morning must not govern the afternoon")
}

func TestCapacityWithNoReadingIsOK(t *testing.T) {
	now := time.Now()
	require.Equal(t, squirrel.CapacityOK,
		squirrel.CapacityOf(squirrel.Checkin{}, false, now),
		"someone who never uses the check-in must not get a smaller product")
}
