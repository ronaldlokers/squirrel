package boot

import (
	"bytes"
	"context"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// What the push path says about itself.
//
// It said nothing at all: no line when it sent, no line about how many
// browsers it found, no line on success. Only a failure spoke. So a send to
// zero subscriptions and a send that worked produced identical logs — which is
// how a feature with no subscribers at all ran in production for weeks without
// anyone being able to tell, and why establishing it took four rounds of
// guessing on the night it was finally chased down.
//
// The rule this encodes: a channel that cannot report having no listeners
// cannot be trusted to report anything.

// logged captures what the push path said, as records rather than as a string,
// so an assertion is about a field and not about phrasing.
func logged(t *testing.T, f func()) []map[string]any {
	t.Helper()
	var buf bytes.Buffer
	was := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(was) })

	f()

	var out []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var rec map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &rec))
		out = append(out, rec)
	}
	return out
}

// Matched on a prefix, so the sentence can be reworded without the test
// pretending that is a behaviour change.
func findMsg(records []map[string]any, msg string) map[string]any {
	for _, r := range records {
		if s, ok := r["msg"].(string); ok && strings.HasPrefix(s, msg) {
			return r
		}
	}
	return nil
}

func TestPushingNobodySaysSo(t *testing.T) {
	records := logged(t, func() {
		sendTo(t, nil)
	})

	rec := findMsg(records, "nobody to push to")
	require.NotNil(t, rec, "a send to an empty list must not look like a send that worked; said: %v", records)
}

func TestPushingSomebodySaysHowManyAndThatItLanded(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer service.Close()

	records := logged(t, func() {
		sendTo(t, []squirrel.Subscription{testSub(t, service.URL)})
	})

	sending := findMsg(records, "pushing")
	require.NotNil(t, sending, "said: %v", records)
	require.Equal(t, float64(1), sending["subscriptions"])

	require.NotNil(t, findMsg(records, "pushed"),
		"a delivery nobody records is a delivery nobody can confirm; said: %v", records)
}

// A store that holds whatever the test wants it to hold, so this can exercise
// the fan-out without a database. What is under test is what the path says,
// not where the rows came from.
type fakeSubs struct {
	live    []squirrel.Subscription
	retired []int64
	kept    []squirrel.Push
}

func (f *fakeSubs) LiveSubscriptions(context.Context, int64) ([]squirrel.Subscription, error) {
	return f.live, nil
}

func (f *fakeSubs) SubscriptionGone(_ context.Context, id int64, _ time.Time) error {
	f.retired = append(f.retired, id)
	return nil
}

func (f *fakeSubs) RecordSaid(_ context.Context, _ int64, p squirrel.Push, _ time.Time) error {
	f.kept = append(f.kept, p)
	return nil
}

func sendTo(t *testing.T, live []squirrel.Subscription) {
	t.Helper()
	push := pusher(testPushCfg(t), &fakeSubs{live: live})
	require.NotNil(t, push)
	require.NoError(t, push(context.Background(), 1, squirrel.Push{Title: "dentist"}))
}

func testPushCfg(t *testing.T) squirrel.PushConfig {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	private := make([]byte, 32)
	key.D.FillBytes(private)
	return squirrel.PushConfig{
		PublicKey:  base64.RawURLEncoding.EncodeToString(elliptic.Marshal(elliptic.P256(), key.X, key.Y)),
		PrivateKey: base64.RawURLEncoding.EncodeToString(private),
		Contact:    "mailto:squirrel@example.invalid",
	}
}

func testSub(t *testing.T, endpoint string) squirrel.Subscription {
	t.Helper()
	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)
	return squirrel.Subscription{
		ID: 7, Endpoint: endpoint + "/push/abc",
		P256dh: base64.RawURLEncoding.EncodeToString(ua.PublicKey().Bytes()),
		Auth:   base64.RawURLEncoding.EncodeToString(auth),
	}
}

func TestWhatWasPushedIsKeptOnce(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer service.Close()

	subs := &fakeSubs{live: []squirrel.Subscription{
		testSub(t, service.URL), testSub(t, service.URL),
	}}
	push := pusher(testPushCfg(t), subs)
	require.NoError(t, push(context.Background(), 1, squirrel.Push{Title: "dentist", Body: "leave now"}))

	require.Len(t, subs.kept, 1, "two browsers on one account kept two rows for one thing said")
	require.Equal(t, "dentist", subs.kept[0].Title)
	require.Equal(t, "leave now", subs.kept[0].Body)
}

func TestNothingIsKeptWhenEveryEndpointRefused(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer service.Close()

	subs := &fakeSubs{live: []squirrel.Subscription{testSub(t, service.URL)}}
	push := pusher(testPushCfg(t), subs)
	require.NoError(t, push(context.Background(), 1, squirrel.Push{Title: "dentist"}))

	require.Empty(t, subs.kept,
		"the app kept a record of saying something no push service would take")
}

func TestNothingIsKeptWhenThereIsNobodyToTell(t *testing.T) {
	subs := &fakeSubs{}
	push := pusher(testPushCfg(t), subs)
	require.NoError(t, push(context.Background(), 1, squirrel.Push{Title: "dentist"}))

	require.Empty(t, subs.kept, "the app kept a record of telling nobody")
}
