package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func TestThePinCarriesTwoVerbsAndNoQuestions(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "the boiler", squirrel.ItemOpen)}}
	deck := opened(t, f, "notes")

	for _, verb := range []string{"drop", "make a chore"} {
		require.Contains(t, deck, verb)
	}
	require.Equal(t, 2, strings.Count(deck, `name="answer" value=`)+strings.Count(deck, `name="chore" value=`),
		"the pin carries something other than the two verbs")
	for _, question := range []string{">done<", ">keep<", "make it a chore", "say it another way", "i can't act on this"} {
		require.NotContains(t, deck, question, "%q is still on the card", question)
	}
}
