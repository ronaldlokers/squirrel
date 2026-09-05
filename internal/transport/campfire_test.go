package transport_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
)

var at = time.Date(2026, 8, 14, 9, 31, 4, 512_000_000, time.UTC)

const payload = `{
  "user":    {"id": 1, "name": "Ronald"},
  "room":    {"id": 7, "name": "Squirrel", "path": "/rooms/7/3-abc/messages"},
  "message": {"id": 42, "body": {"html": "<div>buy milk</div>", "plain": "buy milk"}, "path": "/rooms/7/@42"}
}`

func config() squirrel.CampfireConfig {
	return squirrel.CampfireConfig{
		Path: "/transports/campfire", ConversationID: "7", SenderID: "1",
	}
}

type recordingSink struct {
	outcome squirrel.Outcome
	seen    []squirrel.Capture
}

func (s *recordingSink) Accept(_ context.Context, c squirrel.Capture) squirrel.Outcome {
	s.seen = append(s.seen, c)
	return s.outcome
}

type oneMount struct{ h http.HandlerFunc }

func (m *oneMount) Post(_ string, h http.HandlerFunc) { m.h = h }

func TestCaptureFromMapsThePayload(t *testing.T) {
	got := transport.CaptureFrom([]byte(payload), at)

	require.Equal(t, "campfire", got.Transport)
	require.Equal(t, "42", *got.ExternalID)
	require.Equal(t, "7", *got.ConversationID)
	require.Equal(t, "1", *got.SenderID)
	require.Equal(t, "buy milk", got.Text)
	require.Equal(t, at, got.ReceivedAt)
	require.JSONEq(t, payload, string(got.Payload))
}

func TestCaptureFromKeepsTextVerbatim(t *testing.T) {
	body := `{"message":{"id":42,"body":{"plain":"  DONE  "}}}`
	require.Equal(t, "  DONE  ", transport.CaptureFrom([]byte(body), at).Text)
}

func TestCaptureFromFailsOpenOnUnparseableBody(t *testing.T) {
	got := transport.CaptureFrom([]byte("not json at all"), at)

	require.Nil(t, got.ExternalID)
	require.Nil(t, got.ConversationID)
	require.Nil(t, got.SenderID)
	require.Equal(t, "not json at all", got.Text)
	require.NotEmpty(t, got.Payload)
}

// JSON "null" parses successfully. The TypeScript version crashed here.
func TestCaptureFromFailsOpenOnNullBody(t *testing.T) {
	got := transport.CaptureFrom([]byte("null"), at)
	require.Nil(t, got.ExternalID)
	require.Nil(t, got.ConversationID)
	require.Nil(t, got.SenderID)
}

func TestCaptureFromFailsOpenOnArrayBody(t *testing.T) {
	got := transport.CaptureFrom([]byte("[]"), at)
	require.Nil(t, got.ConversationID)
	require.Nil(t, got.SenderID)
}

func TestCaptureFromFailsOpenOnAChangedShape(t *testing.T) {
	got := transport.CaptureFrom([]byte(`{"message":{"id":42}}`), at)
	require.Nil(t, got.ConversationID)
	require.Nil(t, got.SenderID)
	require.Equal(t, "42", *got.ExternalID)
}

func TestCaptureFromTreatsMissingTextAsEmpty(t *testing.T) {
	require.Empty(t, transport.CaptureFrom([]byte(`{"message":{"id":42}}`), at).Text)
}

func TestRespondStored(t *testing.T) {
	rec := httptest.NewRecorder()
	transport.Respond(rec, squirrel.Stored)

	// The receipt is a boost now. Nothing is posted into the room.
	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("Content-Type"))
	require.Empty(t, rec.Body.String())
}

// Campfire turns any response carrying a Content-Type into a room message, so
// the only way to say nothing is to send no Content-Type at all.
func TestRespondIgnoredSendsNoContentType(t *testing.T) {
	rec := httptest.NewRecorder()
	transport.Respond(rec, squirrel.Ignored)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Empty(t, rec.Header().Get("Content-Type"))
	require.Empty(t, rec.Body.String())
}

func TestRespondFailedIsStill200(t *testing.T) {
	rec := httptest.NewRecorder()
	transport.Respond(rec, squirrel.Failed)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "resend")
}

func TestCampfireStoresACapture(t *testing.T) {
	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}

	stop, err := transport.NewCampfire(config()).Start(context.Background(), sink, mount)
	require.NoError(t, err)
	require.NotNil(t, mount.h)

	rec := httptest.NewRecorder()
	mount.h(rec, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))

	require.Empty(t, rec.Header().Get("Content-Type"))
	require.Empty(t, rec.Body.String())
	require.Len(t, sink.seen, 1)
	require.Equal(t, "buy milk", sink.seen[0].Text)

	require.NoError(t, stop(context.Background()))
}

func TestCampfireLetsABurstOfRapidTypingThrough(t *testing.T) {
	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}

	_, err := transport.NewCampfire(config()).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	for i := range 5 {
		rec := httptest.NewRecorder()
		mount.h(rec, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))
		require.NotEqual(t, http.StatusTooManyRequests, rec.Code, "request %d of an ordinary burst was throttled", i)
	}
}

func TestCampfireRateLimitsASustainedFlood(t *testing.T) {
	mount := &oneMount{}
	sink := &recordingSink{outcome: squirrel.Stored}

	_, err := transport.NewCampfire(config()).Start(context.Background(), sink, mount)
	require.NoError(t, err)

	var throttled *httptest.ResponseRecorder
	for range 200 {
		rec := httptest.NewRecorder()
		mount.h(rec, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))
		if rec.Code == http.StatusTooManyRequests {
			throttled = rec
			break
		}
	}
	require.NotNil(t, throttled, "200 requests with no pause were all accepted")
	require.Empty(t, throttled.Header().Get("Content-Type"),
		"a throttled response carried a Content-Type, which Campfire uploads as a file")
	require.Empty(t, throttled.Body.String())
}

type panickingSink struct{}

func (panickingSink) Accept(context.Context, squirrel.Capture) squirrel.Outcome {
	panic("sink exploded")
}

// Sink is supplied by the caller of NewCampfire, not written by this
// package. A panicking Sink must still answer Campfire — nothing may throw
// out of the handler — because unwinding into the server's own recover
// answers with a bare 500 and no Content-Type, the one response Campfire
// treats as silence, so the sender never learns their thought was dropped.
func TestCampfireNeverPanicsOutOfTheHandlerEvenWhenTheSinkDoes(t *testing.T) {
	mount := &oneMount{}

	stop, err := transport.NewCampfire(config()).Start(context.Background(), panickingSink{}, mount)
	require.NoError(t, err)
	require.NotNil(t, mount.h)

	rec := httptest.NewRecorder()
	require.NotPanics(t, func() {
		mount.h(rec, httptest.NewRequest(http.MethodPost, "/transports/campfire", strings.NewReader(payload)))
	})

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), "resend")

	require.NoError(t, stop(context.Background()))
}

func TestCampfirePayloadIsValidJSONForAnUnparseableBody(t *testing.T) {
	got := transport.CaptureFrom([]byte("not json"), at)
	var anything any
	require.NoError(t, json.Unmarshal(got.Payload, &anything))
}
