package squirrel_test

import (
	"testing"
	"time"

	_ "time/tzdata"

	"github.com/stretchr/testify/require"
)

// Guards against tzdata missing from the image. Without it LoadLocation errors
// or UTC is silently substituted, and every digest lands at the wrong hour.
func TestAmsterdamKnowsAboutSummer(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	july := time.Date(2026, 7, 1, 12, 0, 0, 0, loc)
	_, offset := july.Zone()
	require.Equal(t, 2*60*60, offset, "CEST is UTC+2")

	january := time.Date(2026, 1, 1, 12, 0, 0, 0, loc)
	_, offset = january.Zone()
	require.Equal(t, 1*60*60, offset, "CET is UTC+1")
}

func TestDSTDaysAreNot24Hours(t *testing.T) {
	loc, err := time.LoadLocation("Europe/Amsterdam")
	require.NoError(t, err)

	for _, c := range []struct {
		date  string
		hours float64
	}{
		{"2026-03-29", 23},
		{"2026-10-25", 25},
	} {
		day, err := time.ParseInLocation("2006-01-02", c.date, loc)
		require.NoError(t, err)
		next := day.AddDate(0, 0, 1)
		require.Equal(t, c.hours, next.Sub(day).Hours(), c.date)
	}
}
