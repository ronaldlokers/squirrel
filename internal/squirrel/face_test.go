package squirrel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEveryMoodWearsItsTwoDrawingsAndNoOther(t *testing.T) {
	start := time.Date(2026, 8, 22, 9, 0, 0, 0, time.UTC)

	for _, m := range Moods {
		seen := map[string]bool{}
		for d := 0; d < 14; d++ {
			seen[Face(m, start.AddDate(0, 0, d))] = true
		}
		require.Equal(t, map[string]bool{
			"mood-" + string(m) + ".png":   true,
			"mood-" + string(m) + "-2.png": true,
		}, seen, "%s in a fortnight", m)
	}
}
