package squirrel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

type brokenWriter struct{}

func (brokenWriter) Write(squirrel.Capture) (string, error) {
	return "", errors.New("no space left on device")
}

type panickingWriter struct{}

func (panickingWriter) Write(squirrel.Capture) (string, error) {
	panic("writer exploded")
}

func TestSinkStoresAnAcceptedCapture(t *testing.T) {
	sp, _ := openSpool(t)
	sink := squirrel.NewSink(sp, allows, squirrel.SinkHooks{})

	require.Equal(t, squirrel.Stored, sink.Accept(context.Background(), capture(nil)))

	names, err := sp.List()
	require.NoError(t, err)
	require.Len(t, names, 1)
}

func TestSinkIgnoresAndWritesNothing(t *testing.T) {
	sp, _ := openSpool(t)
	sink := squirrel.NewSink(sp, allows, squirrel.SinkHooks{})

	c := capture(func(c *squirrel.Capture) { c.ConversationID = squirrel.Ptr("8") })
	require.Equal(t, squirrel.Ignored, sink.Accept(context.Background(), c))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

// Silence in the room, but not silence in the logs.
func TestSinkReportsAnIgnoredConversation(t *testing.T) {
	sp, _ := openSpool(t)
	var seen *squirrel.Capture
	sink := squirrel.NewSink(sp, allows, squirrel.SinkHooks{
		OnIgnored: func(c squirrel.Capture) { seen = &c },
	})

	c := capture(func(c *squirrel.Capture) { c.ConversationID = squirrel.Ptr("8") })
	sink.Accept(context.Background(), c)

	require.NotNil(t, seen)
	require.Equal(t, "8", *seen.ConversationID)
}

func TestSinkStoresAFailOpenCapture(t *testing.T) {
	sp, _ := openSpool(t)
	sink := squirrel.NewSink(sp, allows, squirrel.SinkHooks{})

	c := capture(func(c *squirrel.Capture) {
		c.ConversationID = nil
		c.SenderID = nil
	})
	require.Equal(t, squirrel.Stored, sink.Accept(context.Background(), c))
}

// The one path where the bot admits it could not do its job.
func TestSinkReportsFailureRatherThanPretending(t *testing.T) {
	var got error
	sink := squirrel.NewSink(brokenWriter{}, allows, squirrel.SinkHooks{
		OnError: func(err error) { got = err },
	})

	require.Equal(t, squirrel.Failed, sink.Accept(context.Background(), capture(nil)))
	require.Error(t, got)
}

// A panicking writer must not take the process down with it. Accept always
// returns an Outcome.
func TestSinkSurvivesAPanickingWriter(t *testing.T) {
	sink := squirrel.NewSink(panickingWriter{}, allows, squirrel.SinkHooks{})
	require.Equal(t, squirrel.Failed, sink.Accept(context.Background(), capture(nil)))
}

// A hook is observability. It must not be able to break the path it observes.
func TestSinkSurvivesAPanickingHook(t *testing.T) {
	sp, _ := openSpool(t)
	sink := squirrel.NewSink(sp, allows, squirrel.SinkHooks{
		OnIgnored: func(squirrel.Capture) { panic("hook exploded") },
	})

	c := capture(func(c *squirrel.Capture) { c.ConversationID = squirrel.Ptr("8") })
	require.Equal(t, squirrel.Ignored, sink.Accept(context.Background(), c))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}
