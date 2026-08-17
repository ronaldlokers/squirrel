package transport_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/transport"
)

const actionBody = `{
  "type": "action",
  "room": { "id": 9, "name": "Squirrel" },
  "user": { "id": 1, "name": "Ronald" },
  "message": { "id": 451 },
  "action": { "value": "done:2", "selected": true }
}`

func TestCaptureFromReadsAnAction(t *testing.T) {
	at := time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC)
	c := transport.CaptureFrom([]byte(actionBody), at)

	require.Equal(t, "campfire", c.Transport)
	require.Equal(t, "9", *c.ConversationID)
	require.Equal(t, "1", *c.SenderID)
	require.Equal(t, "!action 451 done:2 true", c.Text)
	require.Contains(t, *c.ExternalID, "action:451:1:done:2:true",
		"stable across a retry within the same instant")
}

// Two genuine taps must not collapse into one row. The payload has no event id
// and no timestamp, so ours is the only thing that can separate them.
func TestTwoTapsGetDistinctExternalIDs(t *testing.T) {
	first := transport.CaptureFrom([]byte(actionBody), time.Date(2026, 8, 17, 9, 0, 0, 0, time.UTC))
	second := transport.CaptureFrom([]byte(actionBody), time.Date(2026, 8, 17, 9, 0, 1, 0, time.UTC))
	require.NotEqual(t, *first.ExternalID, *second.ExternalID)
}

// An absent type is a message: that is what an upstream Campfire sends, and
// phase 3 must degrade rather than break.
func TestAbsentTypeIsStillAMessage(t *testing.T) {
	c := transport.CaptureFrom([]byte(`{"room":{"id":9},"user":{"id":1},
		"message":{"id":7,"body":{"plain":"buy milk"}}}`), time.Now())
	require.Equal(t, "buy milk", c.Text)
	require.Equal(t, "7", *c.ExternalID)
	require.False(t, strings.HasPrefix(c.Text, "!action"))
}
