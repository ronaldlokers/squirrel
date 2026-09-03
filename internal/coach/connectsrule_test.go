package coach

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// The clause may point at the board, and the four things it may not do.
func TestTheClauseIsToldWhatItMayPointAtAndWhatItMayNot(t *testing.T) {
	for _, said := range []string{
		"You cannot choose any of them",
		"the detail this thing needs is written down",
		"the same errand",
		"Only when it is true",
		"Never count anything",
		"never say anything about the person",
	} {
		require.Contains(t, decidePreamble, said,
			"the preamble stopped saying %q", said)
	}
}

// It stays one lower-case clause: a connection is a thing to say in the clause,
// not a reason to be given a second sentence.
func TestTheClauseIsStillOneClause(t *testing.T) {
	require.Contains(t, decidePreamble, "lower case, one clause, no full stop")
}
