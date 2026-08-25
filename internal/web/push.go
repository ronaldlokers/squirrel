package web

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Subscribing a browser, which is the only part of Web Push that happens here.
// The encryption and the signing live in the core beside the specification
// they implement.

// subscribeLimit is a guard rather than a rule. An endpoint is a URL and the
// two keys are fixed-length base64; anything much larger is not a
// subscription.
const subscribeLimit = 4 << 10

// pushSubscribeHandler stores where to reach this browser.
//
// It takes the subscription object the browser hands back verbatim, because
// re-shaping it here would be a second place for the field names to be wrong.
// Behind the same origin check as every other write: the identity says who is
// asking, sameOrigin says which page asked.
func pushSubscribeHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}

		var body struct {
			Endpoint string `json:"endpoint"`
			Keys     struct {
				P256dh string `json:"p256dh"`
				Auth   string `json:"auth"`
			} `json:"keys"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, subscribeLimit)).Decode(&body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// A push endpoint is an https URL at a service the browser chose. This
		// is not a permission check — the identity already made one — it is a
		// refusal to store something that could never be sent to.
		if !strings.HasPrefix(body.Endpoint, "https://") ||
			body.Keys.P256dh == "" || body.Keys.Auth == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := s.SaveSubscription(r.Context(), personID, squirrel.Subscription{
			Endpoint: body.Endpoint,
			P256dh:   body.Keys.P256dh,
			Auth:     body.Keys.Auth,
		}); err != nil {
			fail(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
