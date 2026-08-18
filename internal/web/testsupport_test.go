package web

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// fakeStore is an in-memory pile. The screen's own tests must not need
// Postgres: what is under test here is routing, rendering and the refusal to
// count, none of which is a database question. The store's own behaviour is
// covered by the integration tests in internal/squirrel.
type fakeStore struct {
	items []squirrel.Item
	err   error
}

func (f *fakeStore) OpenItems(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if it.State == squirrel.ItemOpen {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

func (f *fakeStore) SearchItems(_ context.Context, _ int64, q string, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if strings.Contains(strings.ToLower(it.RawText), strings.ToLower(q)) {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

func (f *fakeStore) ItemByID(_ context.Context, _ int64, id int64) (squirrel.Item, bool, error) {
	if f.err != nil {
		return squirrel.Item{}, false, f.err
	}
	for _, it := range f.items {
		if it.ID == id {
			return it, true, nil
		}
	}
	return squirrel.Item{}, false, nil
}

func (f *fakeStore) SetItemState(_ context.Context, id int64, state squirrel.ItemState, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].State = state
		}
	}
	return nil
}

func (f *fakeStore) PromoteItem(_ context.Context, _ int64, id int64, _ time.Duration) (squirrel.Chore, bool, error) {
	if f.err != nil {
		return squirrel.Chore{}, false, f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].State = squirrel.ItemDone
			return squirrel.Chore{Name: f.items[i].RawText}, true, nil
		}
	}
	return squirrel.Chore{}, false, nil
}

// testMux collects routes so a test can call one directly.
type testMux struct{ routes map[string]http.HandlerFunc }

func newTestMux() *testMux { return &testMux{routes: map[string]http.HandlerFunc{}} }

func (m *testMux) Get(pattern string, h http.HandlerFunc)  { m.routes["GET "+pattern] = h }
func (m *testMux) Post(pattern string, h http.HandlerFunc) { m.routes["POST "+pattern] = h }

func (m *testMux) call(t *testing.T, method, target string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	// Longest pattern first, so /pile/act is not answered by /pile. A map
	// iterates in a random order and the real ServeMux resolves this by
	// specificity; this helper has to do the same or the test is a coin toss.
	best := ""
	for pattern := range m.routes {
		wantMethod, path, _ := strings.Cut(pattern, " ")
		if wantMethod != method || !strings.HasPrefix(target, path) {
			continue
		}
		if len(pattern) > len(best) {
			best = pattern
		}
	}
	if best == "" {
		t.Fatalf("no route for %s %s", method, target)
		return nil
	}

	r := httptest.NewRequest(method, target, body)
	r.Header.Set("X-Authentik-Username", "ronald")
	if method == "POST" {
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		// What a browser sends when this screen's own form is submitted.
		// csrf_test.go covers what it sends when someone else's is.
		r.Header.Set("Origin", "http://"+r.Host)
	}
	w := httptest.NewRecorder()
	m.routes[best](w, r)
	return w
}

func note(id int64, text string, state squirrel.ItemState) squirrel.Item {
	return squirrel.Item{ID: id, RawText: text, ReceivedAt: time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC), State: state}
}

func mounted(t *testing.T, f *fakeStore) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		Path: "/pile", IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 },
	}))
	return m
}
