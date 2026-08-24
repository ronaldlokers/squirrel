package squirrel_test

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The keys on the wire are the ones the service worker reads.
//
// This is a contract with a file in another language, and the existing test
// could not see it: it decrypted the payload and unmarshalled it back into
// squirrel.Push, so it round-tripped through the same capitalisation the bug is
// made of and passed. Go agreed with Go.
//
// What sw.js actually does is
//
//	said = { title: "Squirrel", body: "" }
//	said = { ...said, ...event.data.json() }
//	showNotification(said.title, { body: said.body })
//
// so a payload of {"Title":…,"Body":…} adds two keys nobody reads and leaves
// the defaults in place. Every leave-by warning that arrived would have been a
// notification saying "Squirrel", with no words in it, about nothing.
//
// Asserted against a map rather than a struct, because a struct is the thing
// that hid it.
func TestThePushPayloadUsesTheKeysTheBrowserReads(t *testing.T) {
	uaPrivate, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	authSecret := make([]byte, 16)
	_, err = rand.Read(authSecret)
	require.NoError(t, err)

	var got []byte
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
	}))
	defer service.Close()

	_, err = squirrel.SendPush(context.Background(), service.Client(), testPushConfig(t),
		squirrel.Subscription{
			Endpoint: service.URL + "/push/abc",
			P256dh:   b64.EncodeToString(uaPrivate.PublicKey().Bytes()),
			Auth:     b64.EncodeToString(authSecret),
		},
		squirrel.Push{Title: "dentist", Body: "leave about 14:10", URL: "/"})
	require.NoError(t, err)

	var wire map[string]any
	require.NoError(t, json.Unmarshal(decryptAsBrowser(t, got, uaPrivate, authSecret), &wire))

	require.Equal(t, "dentist", wire["title"],
		`sw.js reads said.title; anything else leaves the notification saying "Squirrel"`)
	require.Equal(t, "leave about 14:10", wire["body"],
		"sw.js reads said.body; anything else leaves the notification empty")
	require.Equal(t, "/", wire["url"])

	for _, capitalised := range []string{"Title", "Body", "URL"} {
		require.NotContains(t, wire, capitalised,
			"a key the browser does not read is a key that does nothing")
	}
}
