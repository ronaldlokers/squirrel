package squirrel

import (
	"bytes"
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
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Web Push, written against the standard library.
//
// This is the one piece of protocol in the product, and it is here rather than
// behind a dependency because the whole of it is RFC 8291's key derivation and
// RFC 8292's signed header — about a hundred lines of standard-library crypto
// against two specifications that are finished and will not move. A library
// would be a supply-chain surface and a version to track for code that never
// changes.
//
// What it is for: "leave about 14:10" has one useful minute. A chat message
// you notice at 14:40 is worse than none, because it teaches you not to trust
// the next one. Nothing else pushes — a nudge is a suggestion, and a
// suggestion that waits is a suggestion doing its job.

// PushConfig is the VAPID identity. Empty means no pushing, which is a
// supported state: the leave-by message still reaches the room, and the screen
// simply never offers to subscribe.
type PushConfig struct {
	// PublicKey is the uncompressed P-256 point, base64url, as the browser
	// needs it for applicationServerKey.
	PublicKey string
	// PrivateKey is the raw 32-byte scalar, base64url. It comes from the
	// secret store and never from the repository.
	PrivateKey string
	// Contact is the mailto: or https: the push service can complain to. Part
	// of the specification rather than a courtesy — services reject a token
	// without one.
	Contact string
}

// Enabled reports whether pushing is configured at all.
func (c PushConfig) Enabled() bool {
	return c.PublicKey != "" && c.PrivateKey != "" && c.Contact != ""
}

// Subscription is one browser.
type Subscription struct {
	ID       int64
	Endpoint string
	P256dh   string
	Auth     string
}

// SaveSubscription stores or refreshes one.
//
// Upsert on the endpoint, because a browser that re-subscribes hands back the
// same one and two rows would mean two copies of every message. Re-subscribing
// also clears gone_at: a browser that is talking to us again is not gone,
// whatever it did last week.
func (s *Store) SaveSubscription(ctx context.Context, personID int64, sub Subscription) error {
	if _, err := s.pool.Exec(ctx, `
		insert into push_subscriptions (person_id, endpoint, p256dh, auth)
		values ($1, $2, $3, $4)
		on conflict (endpoint) do update
		  set person_id = excluded.person_id,
		      p256dh = excluded.p256dh,
		      auth = excluded.auth,
		      gone_at = null`,
		personID, sub.Endpoint, sub.P256dh, sub.Auth); err != nil {
		return fmt.Errorf("saving a push subscription: %w", err)
	}
	return nil
}

// LiveSubscriptions is every browser still worth sending to.
func (s *Store) LiveSubscriptions(ctx context.Context, personID int64) ([]Subscription, error) {
	rows, err := s.pool.Query(ctx, `
		select id, endpoint, p256dh, auth from push_subscriptions
		 where person_id = $1 and gone_at is null`, personID)
	if err != nil {
		return nil, fmt.Errorf("reading push subscriptions: %w", err)
	}
	defer rows.Close()

	subs := []Subscription{}
	for rows.Next() {
		var sub Subscription
		if err := rows.Scan(&sub.ID, &sub.Endpoint, &sub.P256dh, &sub.Auth); err != nil {
			return nil, fmt.Errorf("scanning a push subscription: %w", err)
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}

// SubscriptionGone marks one dead. A browser that has been uninstalled answers
// 404 or 410 forever, and retrying it makes every later send slower for
// nothing.
func (s *Store) SubscriptionGone(ctx context.Context, id int64, at time.Time) error {
	if _, err := s.pool.Exec(ctx,
		`update push_subscriptions set gone_at = $2 where id = $1`, id, at); err != nil {
		return fmt.Errorf("retiring a push subscription: %w", err)
	}
	return nil
}

// Push is what a message looks like on the other side. Two lines, because a
// notification is read at arm's length in one glance and a third is one nobody
// reads.
//
// The tags are the contract with `sw.js`, which reads `said.title` and
// `said.body` after spreading this object over its own defaults. Without them
// Go marshals `Title` and `Body`, the spread adds two keys nobody reads, and
// the defaults survive — so every notification that arrived said "Squirrel",
// with no words in it, about nothing.
//
// The test that covered this decrypted the payload back into this struct, so it
// round-tripped through the same capitalisation and passed. See
// pushwire_test.go, which asserts the keys on the wire instead.
type Push struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	// URL is where pressing it goes. Always the screen's front door: the offer
	// is there, and a deep link to something that has since been done is worse
	// than a page that says what is true now.
	//
	// `sw.js` does not read it — the click handler navigates to "/" itself. It
	// is tagged with the others so the payload has one spelling convention
	// rather than two, and so a worker that ever does read it finds what it
	// expects.
	URL string `json:"url"`
}

// pushRecordSize is the aes128gcm record size, in bytes. One record is all this
// ever sends: the payload is two short lines.
const pushRecordSize = 4096

// pushTTL is how long a push service may hold a message, in seconds. Long
// enough to survive a phone asleep in a pocket, short enough that a warning
// about leaving never arrives after the thing it was about — a leave-by message
// that shows up tomorrow morning is the one failure worse than silence.
const pushTTL = "600"

// Pusher sends to every live browser. It is a func rather than an interface so
// boot can hand one in without this package learning about HTTP clients.
type Pusher func(ctx context.Context, personID int64, p Push) error

// SendPush encrypts and delivers one push to one subscription.
//
// Every failure is the caller's to swallow. A push that cannot be sent must
// never turn a message that already reached the room into an error — the
// room is the channel that always works, and this is the one that is fast.
func SendPush(ctx context.Context, client *http.Client, cfg PushConfig, sub Subscription, p Push) (gone bool, err error) {
	body, err := json.Marshal(p)
	if err != nil {
		return false, fmt.Errorf("encoding a push: %w", err)
	}

	encrypted, asPublic, salt, err := encryptPush(sub, body)
	if err != nil {
		return false, err
	}

	// The aes128gcm content-encoding header: salt, record size, and the
	// sender's public key inline, so the browser can derive the same key
	// without any prior arrangement. See RFC 8188 §2.1.
	var payload bytes.Buffer
	payload.Write(salt)
	_ = binary.Write(&payload, binary.BigEndian, uint32(pushRecordSize))
	payload.WriteByte(byte(len(asPublic)))
	payload.Write(asPublic)
	payload.Write(encrypted)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.Endpoint, &payload)
	if err != nil {
		return false, fmt.Errorf("building a push request: %w", err)
	}
	token, err := vapidToken(cfg, sub.Endpoint)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", token)
	req.Header.Set("Content-Encoding", "aes128gcm")
	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("TTL", pushTTL)
	req.Header.Set("Urgency", "high")

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("sending a push: %w", err)
	}
	defer resp.Body.Close()

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The browser is gone for good, and the caller should stop trying.
		return true, nil
	case resp.StatusCode >= 300:
		return false, fmt.Errorf("push refused with %s", resp.Status)
	}
	return false, nil
}

// encryptPush is RFC 8291's derivation, in the order the specification gives
// it. The names are the specification's names on purpose: this is the one
// place in the codebase where matching a document beats reading well on its
// own.
func encryptPush(sub Subscription, plaintext []byte) (ciphertext, asPublic, salt []byte, err error) {
	uaPublicBytes, err := b64.DecodeString(sub.P256dh)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding the browser's key: %w", err)
	}
	authSecret, err := b64.DecodeString(sub.Auth)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("decoding the browser's secret: %w", err)
	}

	curve := ecdh.P256()
	uaPublic, err := curve.NewPublicKey(uaPublicBytes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading the browser's key: %w", err)
	}
	asPrivate, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generating a key: %w", err)
	}
	asPublic = asPrivate.PublicKey().Bytes()

	shared, err := asPrivate.ECDH(uaPublic)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("agreeing a key: %w", err)
	}

	// key_info = "WebPush: info" || 0x00 || ua_public || as_public
	keyInfo := append([]byte("WebPush: info\x00"), uaPublicBytes...)
	keyInfo = append(keyInfo, asPublic...)
	ikm, err := hkdf.Key(sha256.New, shared, authSecret, string(keyInfo), 32)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("deriving the key material: %w", err)
	}

	salt = make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return nil, nil, nil, fmt.Errorf("generating a salt: %w", err)
	}
	cek, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: aes128gcm\x00", 16)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("deriving the content key: %w", err)
	}
	nonce, err := hkdf.Key(sha256.New, ikm, salt, "Content-Encoding: nonce\x00", 12)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("deriving the nonce: %w", err)
	}

	block, err := aes.NewCipher(cek)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("preparing the cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("preparing the cipher: %w", err)
	}
	// 0x02 is the delimiter that says "this is the last record". One record is
	// all this ever sends: the longest thing here is two short lines.
	padded := append(append([]byte{}, plaintext...), 0x02)
	return aead.Seal(nil, nonce, padded, nil), asPublic, salt, nil
}

// vapidToken is RFC 8292's Authorization header: a JWT saying who is sending,
// signed with the key the browser was told to expect.
func vapidToken(cfg PushConfig, endpoint string) (string, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("reading the push endpoint: %w", err)
	}

	// Twelve hours, which is the specification's maximum. A shorter one buys
	// nothing here: the token is minted per send and never stored.
	claims := map[string]any{
		"aud": u.Scheme + "://" + u.Host,
		"exp": time.Now().Add(12 * time.Hour).Unix(),
		"sub": cfg.Contact,
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("building a push token: %w", err)
	}

	signing := b64.EncodeToString([]byte(`{"typ":"JWT","alg":"ES256"}`)) + "." +
		b64.EncodeToString(claimsJSON)

	key, err := vapidKey(cfg.PrivateKey)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, key, sum[:])
	if err != nil {
		return "", fmt.Errorf("signing a push token: %w", err)
	}
	// Raw r||s, fixed width. ASN.1 is what SignASN1 would give and is not what
	// JWS ES256 is: a variable-length signature here is rejected by every push
	// service, and the padding is why this is done by hand.
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	return "vapid t=" + signing + "." + b64.EncodeToString(sig) + ", k=" + cfg.PublicKey, nil
}

// vapidKey reads the raw 32-byte scalar the secret store holds and rebuilds
// the key pair around it.
func vapidKey(private string) (*ecdsa.PrivateKey, error) {
	raw, err := b64.DecodeString(strings.TrimSpace(private))
	if err != nil {
		return nil, fmt.Errorf("decoding the push key: %w", err)
	}
	if len(raw) != 32 {
		return nil, fmt.Errorf("the push key must be 32 bytes, got %d", len(raw))
	}
	key := &ecdsa.PrivateKey{D: new(big.Int).SetBytes(raw)}
	key.PublicKey.Curve = elliptic.P256()
	key.PublicKey.X, key.PublicKey.Y = key.PublicKey.Curve.ScalarBaseMult(raw)
	return key, nil
}

// b64 is base64url without padding, which is what every part of this
// specification uses and what the browser hands back.
var b64 = base64.RawURLEncoding
