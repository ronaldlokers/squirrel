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
		{},
	} {
		f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
		code := crossSite(t, mounted(t, f), "/board/act",
			url.Values{"what": {"note"}, "id": {"1"}, "answer": {"done"}}, headers)

		require.Equal(t, 403, code, "headers %v", headers)
		require.Equal(t, squirrel.ItemOpen, f.items[0].State,
			"a cross-site write must not reach the store")
	}
}

func TestPromotionFromAnotherSiteIsRefused(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "bins out", squirrel.ItemOpen)}}
	code := crossSite(t, mounted(t, f), "/board/chore",
		url.Values{"id": {"1"}, "every": {"7"}},
		map[string]string{"Origin": "https://evil.example"})

	require.Equal(t, 403, code)
	require.Equal(t, squirrel.ItemOpen, f.items[0].State)
}

func TestARefererFromThisScreenIsAccepted(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	code := crossSite(t, mounted(t, f), "/board/act",
		url.Values{"what": {"note"}, "id": {"1"}, "answer": {"done"}},
		map[string]string{"Referer": "http://example.com/"})

	require.Equal(t, 303, code)
	require.Equal(t, squirrel.ItemDone, f.items[0].State)
}
