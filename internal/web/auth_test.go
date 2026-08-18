package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func testOptions() Options {
	return Options{
		Path:           "/pile",
		IdentityHeader: "X-Authentik-Username",
		Identity:       "ronald",
		PersonID:       1,
	}
}

func TestGuardAllowsTheConfiguredIdentity(t *testing.T) {
	reached := false
	h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

	r := httptest.NewRequest("GET", "/pile", nil)
	r.Header.Set("X-Authentik-Username", "ronald")
	w := httptest.NewRecorder()
	h(w, r)

	require.True(t, reached)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestGuardRefusesEveryoneElse(t *testing.T) {
	for _, name := range []string{"", "someone", "Ronald "} {
		reached := false
		h := guard(testOptions(), func(http.ResponseWriter, *http.Request) { reached = true })

		r := httptest.NewRequest("GET", "/pile", nil)
		if name != "" {
			r.Header.Set("X-Authentik-Username", name)
		}
		w := httptest.NewRecorder()
		h(w, r)

		require.False(t, reached, "handler ran for identity %q", name)
		require.Equal(t, http.StatusForbidden, w.Code, "identity %q", name)
		require.Empty(t, w.Body.String(), "a refusal says nothing about what is behind it")
	}
}
