package web

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// crossSite posts a form the way another site would: the identity header is
// there, because the browser sends it on every request to this host, and the
// origin is somewhere else.
func crossSite(t *testing.T, m *testMux, path string, form url.Values, headers map[string]string) int {
	t.Helper()
	r := httptest.NewRequest("POST", path, strings.NewReader(form.Encode()))
	r.AddCookie(&http.Cookie{Name: sessionCookie, Value: "a-token"})
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	m.routes["POST "+path](w, r)
	return w.Code
}

func TestAWriteFromAnotherSiteIsRefused(t *testing.T) {
	for _, headers := range []map[string]string{
		{"Origin": "https://evil.example"},
		{"Referer": "https://evil.example/page"},
		{}, // a request that says nothing about where it came from
	} {
		f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
		code := crossSite(t, mounted(t, f), "/pile/act",
			url.Values{"id": {"1"}, "act": {"done"}}, headers)

		require.Equal(t, 403, code, "headers %v", headers)
		require.Equal(t, squirrel.ItemOpen, f.items[0].State,
			"a cross-site write must not reach the store")
	}
}

func TestPromotionFromAnotherSiteIsRefused(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	code := crossSite(t, mounted(t, f), "/pile/chore",
		url.Values{"id": {"1"}, "every": {"every week"}},
		map[string]string{"Origin": "https://evil.example"})

	require.Equal(t, 403, code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State)
}

// Referer is the fallback for a browser that omits Origin on its own form.
func TestARefererFromThisScreenIsAccepted(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	code := crossSite(t, mounted(t, f), "/pile/act",
		url.Values{"id": {"1"}, "act": {"done"}},
		map[string]string{"Referer": "http://example.com/pile"})

	require.Equal(t, 303, code)
	require.Equal(t, squirrel.ItemDone, f.items[0].State)
}
