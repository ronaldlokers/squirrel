package coach_test

import (
	"context"
	"strings"
	"testing"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/stretchr/testify/require"
)

func TestTheRecordHoldsWhatWasOnScreenAndNotOnlyWhatWasTyped(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	log := &fakeLog{}
	p := providerFor(api, log)

	_, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1, Kind: "chat", Room: "everything",
		Now:     coach.Now{Clock: "14:00"},
		Subject: "the boiler service code is 4471",
		Said:    "what was that about",
	})
	require.NoError(t, err)

	require.Len(t, log.recorded, 1)
	kept := log.recorded[0].Prompt

	require.Contains(t, kept, "the boiler service code is 4471",
		"the record kept forever does not contain the content that left the machine")
	require.Contains(t, kept, "what was that about")

	sent, ok := api.messages(t, 0)[1]["content"].(string)
	require.True(t, ok)
	require.True(t, strings.HasPrefix(sent, "On screen: the boiler service code is 4471"),
		"what was sent no longer starts with what is on screen")
	require.True(t, strings.HasPrefix(kept, "On screen: the boiler service code is 4471"),
		"the record and the prompt describe the turn differently")
	require.True(t, strings.HasSuffix(sent, "what was that about"))
	require.True(t, strings.HasSuffix(kept, "what was that about"))
}

func TestTheRecordKeepsNeitherThePreambleNorTheDayItWasSaidOn(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	log := &fakeLog{}
	p := providerFor(api, log)

	_, err := p.Answer(context.Background(), coach.Turn{
		PersonID: 1, Kind: "chat", Room: "everything",
		Now:  coach.Now{Clock: "14:00", Capacity: "low"},
		Said: "what now",
	})
	require.NoError(t, err)

	require.Len(t, log.recorded, 1)
	require.Equal(t, "what now", log.recorded[0].Prompt,
		"the record grew the context line, and a row per turn of state nobody asked it to keep")
}
