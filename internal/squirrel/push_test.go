package squirrel_test

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The one piece of protocol in the product, so it is tested by being a
// browser: the test generates its own key pair, subscribes with it, and
// decrypts what Squirrel sends. Anything less would be checking that the code
// does what the code does.

var b64 = base64.RawURLEncoding

func TestAPushIsSomethingTheBrowserCanDecrypt(t *testing.T) {
	// The browser's half of the subscription.
	uaPrivate, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	authSecret := make([]byte, 16)
	_, err = rand.Read(authSecret)
	require.NoError(t, err)

	var got []byte
	var headers http.Header
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = io.ReadAll(r.Body)
		headers = r.Header.Clone()
		w.WriteHeader(http.StatusCreated)
	}))
	defer service.Close()

	gone, err := squirrel.SendPush(context.Background(), service.Client(), testPushConfig(t),
		squirrel.Subscription{
			Endpoint: service.URL + "/push/abc",
			P256dh:   b64.EncodeToString(uaPrivate.PublicKey().Bytes()),
			Auth:     b64.EncodeToString(authSecret),
		},
		squirrel.Push{Title: "dentist", Body: "at 14:30 — leave about 14:10"})

	require.NoError(t, err)
	require.False(t, gone)
	require.Equal(t, "aes128gcm", headers.Get("Content-Encoding"))
	require.True(t, strings.HasPrefix(headers.Get("Authorization"), "vapid t="))

	plain := decryptAsBrowser(t, got, uaPrivate, authSecret)
	var said squirrel.Push
	require.NoError(t, json.Unmarshal(plain, &said))
	require.Equal(t, "dentist", said.Title)
	require.Contains(t, said.Body, "14:10")
}

// A browser that has been uninstalled says so, and the caller is told to stop
// trying rather than to report an error every minute forever.
func TestAnUninstalledBrowserIsReportedGoneRatherThanFailed(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer service.Close()

	gone, err := squirrel.SendPush(context.Background(), service.Client(), testPushConfig(t),
		testSubscription(t, service.URL), squirrel.Push{Title: "dentist"})

	require.NoError(t, err)
	require.True(t, gone)
}

func TestAPushServiceRefusingIsAnError(t *testing.T) {
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer service.Close()

	gone, err := squirrel.SendPush(context.Background(), service.Client(), testPushConfig(t),
		testSubscription(t, service.URL), squirrel.Push{Title: "dentist"})

	require.Error(t, err)
	require.False(t, gone, "refused is not gone: it is worth trying again")
}

// The signature is raw r||s at a fixed width. ASN.1 is what the standard
// library's other signing function gives and is not what JWS ES256 is — every
// push service rejects the difference, and it is invisible until one does.
func TestTheTokenIsAJWTWithAFixedWidthSignature(t *testing.T) {
	var authorization string
	service := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusCreated)
	}))
	defer service.Close()

	_, err := squirrel.SendPush(context.Background(), service.Client(), testPushConfig(t),
		testSubscription(t, service.URL), squirrel.Push{Title: "dentist"})
	require.NoError(t, err)

	token, _, _ := strings.Cut(strings.TrimPrefix(authorization, "vapid t="), ",")
	parts := strings.Split(strings.TrimSpace(token), ".")
	require.Len(t, parts, 3)

	header, err := b64.DecodeString(parts[0])
	require.NoError(t, err)
	require.Contains(t, string(header), `"ES256"`)

	sig, err := b64.DecodeString(parts[2])
	require.NoError(t, err)
	require.Len(t, sig, 64, "raw r||s, never ASN.1")
}

func TestPushIsOffUntilAllThreeSettingsArePresent(t *testing.T) {
	require.False(t, squirrel.PushConfig{}.Enabled())
	require.False(t, squirrel.PushConfig{PublicKey: "a", PrivateKey: "b"}.Enabled())
	require.True(t, squirrel.PushConfig{PublicKey: "a", PrivateKey: "b", Contact: "c"}.Enabled())
}

// testPushConfig mints a throwaway VAPID pair. Never a fixed one: a key
// literal in a test file is a key that eventually gets copied somewhere real.
func testPushConfig(t *testing.T) squirrel.PushConfig {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	private := make([]byte, 32)
	key.D.FillBytes(private)
	public := elliptic.Marshal(elliptic.P256(), key.X, key.Y)

	return squirrel.PushConfig{
		PublicKey:  b64.EncodeToString(public),
		PrivateKey: b64.EncodeToString(private),
		Contact:    "mailto:squirrel@example.invalid",
	}
}

func testSubscription(t *testing.T, endpoint string) squirrel.Subscription {
	t.Helper()
	ua, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	auth := make([]byte, 16)
	_, err = rand.Read(auth)
	require.NoError(t, err)

	return squirrel.Subscription{
		Endpoint: endpoint + "/push/abc",
		P256dh:   b64.EncodeToString(ua.PublicKey().Bytes()),
		Auth:     b64.EncodeToString(auth),
	}
}

// decryptAsBrowser is RFC 8291 from the receiving side, written out rather
// than shared with the sender: two implementations of the same derivation is
// the only way this test can fail when the sender is wrong.
func decryptAsBrowser(t *testing.T, body []byte, uaPrivate *ecdh.PrivateKey, authSecret []byte) []byte {
	t.Helper()
	require.Greater(t, len(body), 21)

	salt := body[:16]
	require.Equal(t, uint32(4096), binary.BigEndian.Uint32(body[16:20]))
	keyLen := int(body[20])
	asPublicBytes := body[21 : 21+keyLen]
	ciphertext := body[21+keyLen:]

	asPublic, err := ecdh.P256().NewPublicKey(asPublicBytes)
	require.NoError(t, err)
	shared, err := uaPrivate.ECDH(asPublic)
	require.NoError(t, err)

	keyInfo := append([]byte("WebPush: info\x00"), uaPrivate.PublicKey().Bytes()...)
	keyInfo = append(keyInfo, asPublicBytes...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	require.NoError(t, err)

	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	require.NoError(t, err)
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	require.NoError(t, err)

	block, err := aes.NewCipher(cek)
	require.NoError(t, err)
	aead, err := cipher.NewGCM(block)
	require.NoError(t, err)
	plain, err := aead.Open(nil, nonce, ciphertext, nil)
	require.NoError(t, err)

	// The trailing 0x02 says "last record" and is not part of the message.
	require.Equal(t, byte(0x02), plain[len(plain)-1])
	return plain[:len(plain)-1]
}
