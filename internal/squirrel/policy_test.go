package squirrel_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

var allows = []squirrel.Allow{
	{Transport: "campfire", ConversationID: "7", SenderID: "1"},
}

func capture(mutate func(*squirrel.Capture)) squirrel.Capture {
	c := squirrel.Capture{
		Transport:      "campfire",
		ExternalID:     squirrel.Ptr("42"),
		ConversationID: squirrel.Ptr("7"),
		SenderID:       squirrel.Ptr("1"),
		Text:           "buy milk",
		ReceivedAt:     time.Date(2026, 8, 14, 9, 31, 4, 512_000_000, time.UTC),
		Payload:        []byte(`{}`),
	}
	if mutate != nil {
		mutate(&c)
	}
	return c
}

func TestDecideAcceptsConfiguredConversationAndSender(t *testing.T) {
	require.Equal(t, squirrel.Accept, squirrel.Decide(capture(nil), allows))
}

func TestDecideIgnoresAnotherConversation(t *testing.T) {
	c := capture(func(c *squirrel.Capture) { c.ConversationID = squirrel.Ptr("8") })
	require.Equal(t, squirrel.Ignore, squirrel.Decide(c, allows))
}

func TestDecideIgnoresAnotherSender(t *testing.T) {
	c := capture(func(c *squirrel.Capture) { c.SenderID = squirrel.Ptr("2") })
	require.Equal(t, squirrel.Ignore, squirrel.Decide(c, allows))
}

func TestDecideIgnoresUnconfiguredTransport(t *testing.T) {
	c := capture(func(c *squirrel.Capture) { c.Transport = "matrix" })
	require.Equal(t, squirrel.Ignore, squirrel.Decide(c, allows))
}

// The load-bearing case. If Campfire changes its payload shape and room.id
// goes missing, failing closed drops every capture silently for days.
func TestDecideFailsOpenOnUnknownConversation(t *testing.T) {
	c := capture(func(c *squirrel.Capture) { c.ConversationID = nil })
	require.Equal(t, squirrel.Accept, squirrel.Decide(c, allows))
}

func TestDecideFailsOpenOnUnknownSender(t *testing.T) {
	c := capture(func(c *squirrel.Capture) { c.SenderID = nil })
	require.Equal(t, squirrel.Accept, squirrel.Decide(c, allows))
}

func TestDecideFailsOpenEvenForUnconfiguredTransport(t *testing.T) {
	c := capture(func(c *squirrel.Capture) {
		c.Transport = "matrix"
		c.ConversationID = nil
	})
	require.Equal(t, squirrel.Accept, squirrel.Decide(c, allows))
}

// Matching must be against one entry, not three independent matches across
// different entries.
func TestDecideMatchesASingleEntry(t *testing.T) {
	many := append([]squirrel.Allow{}, allows...)
	many = append(many, squirrel.Allow{
		Transport: "matrix", ConversationID: "!room:example", SenderID: "@me:example",
	})

	matrix := capture(func(c *squirrel.Capture) {
		c.Transport = "matrix"
		c.ConversationID = squirrel.Ptr("!room:example")
		c.SenderID = squirrel.Ptr("@me:example")
	})
	require.Equal(t, squirrel.Accept, squirrel.Decide(matrix, many))

	crossed := capture(func(c *squirrel.Capture) {
		c.Transport = "matrix"
		c.ConversationID = squirrel.Ptr("7")
		c.SenderID = squirrel.Ptr("@me:example")
	})
	require.Equal(t, squirrel.Ignore, squirrel.Decide(crossed, many))
}
