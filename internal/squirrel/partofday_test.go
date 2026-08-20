package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestPartOfDayUsesTheSameHoursAskingDoes(t *testing.T) {
	at := func(hour int) time.Time {
		return time.Date(2026, 8, 20, hour, 30, 0, 0, time.UTC)
	}

	for hour, want := range map[int]squirrel.DayPart{
		6:  squirrel.Morning,
		11: squirrel.Morning,
		12: squirrel.Afternoon,
		16: squirrel.Afternoon,
		17: squirrel.Evening,
		21: squirrel.Evening,
		// Nothing asks between 22:00 and 06:00, and nothing here names it
		// either: AnyPart rather than a fifth word the rest of the product
		// does not know.
		22: squirrel.AnyPart,
		3:  squirrel.AnyPart,
	} {
		require.Equal(t, want, squirrel.PartOfDay(at(hour)), "at %02d:30", hour)
	}
}
