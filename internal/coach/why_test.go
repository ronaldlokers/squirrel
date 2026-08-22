package coach

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

// Every reason Buddy goes quiet collapses to one behaviour on purpose — the
// picker chooses, the ladder answers, and nothing about the product stops.
// That is Rule 10 and it stays.
//
// The reasons are not one thing, though, and the person who runs this branches
// on them: a retired model id in prices.go reads exactly like a five-second
// network blip in the log, and the two want opposite things done about them.
// Buddy could be broken for a fortnight with nothing surfacing it beyond
// replies getting blander.
func TestTheLogSaysWhyTheProviderFailed(t *testing.T) {
	for _, tc := range []struct {
		name string
		hand func(w http.ResponseWriter, r *http.Request)
		why  string
	}{
		{"a retired model or a revoked key", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":{"message":"the model does not exist"}}`))
		}, "refused"},
		{"a proxy in front of it", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte("<html>sign in</html>"))
		}, "nonsense"},
		{"an answer with no completion in it", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"choices":[]}`))
		}, "nonsense"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(tc.hand))
			defer srv.Close()

			p := &Provider{BaseURL: srv.URL, APIKey: "x", Client: srv.Client()}
			_, _, _, _, err := p.completionWithTools(t.Context(), Permit{}, "m", nil, nil)

			require.Error(t, err)
			require.True(t, errors.Is(err, ErrUnavailable),
				"a caller must still see one error and one only")
			require.Equal(t, tc.why, Why(err),
				"the log cannot tell this apart from any other silence")
		})
	}
}

// The provider not answering at all is its own reason, and the one that means
// "wait".
func TestAProviderThatDoesNotAnswerSaysSo(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	srv.Close() // nothing is listening now

	p := &Provider{BaseURL: srv.URL, APIKey: "x", Client: srv.Client()}
	_, _, _, _, err := p.completionWithTools(t.Context(), Permit{}, "m", nil, nil)

	require.Error(t, err)
	require.Equal(t, "unreachable", Why(err))
}

// And an error that is none of these says nothing rather than guessing.
func TestAnErrorWithNoReasonNamesNone(t *testing.T) {
	require.Empty(t, Why(errors.New("something else entirely")))
	require.Empty(t, Why(nil))
}
