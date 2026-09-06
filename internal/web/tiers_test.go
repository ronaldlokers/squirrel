package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestTheCardCarriesFourVerbsAndNoQuestions(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)}}
	deck := opened(t, f, "notes")

	for _, verb := range []string{"done", "keep", "drop", "make a chore"} {
		require.Contains(t, deck, verb)
	}
	require.Equal(t, 4, strings.Count(deck, `name="answer" value=`)+strings.Count(deck, `name="chore" value=`),
		"the strip carries something other than the four verbs")
	for _, question := range []string{"make it a chore", "say it another way", "i can't act on this"} {
		require.NotContains(t, deck, question, "%q is still on the card", question)
	}
}
