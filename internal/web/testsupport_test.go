package web

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
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
	items   []squirrel.Item
	chores  []squirrel.Chore
	checkin *squirrel.Checkin
	// readings is the fortnight the moods page reads, newest first.
	readings []squirrel.Checkin
	timer    *squirrel.Timer
	err      error

	// offer is what the picker hands back, and nil is "nothing to hand you" —
	// the case the screen has to render as an absence rather than as an empty
	// box. gated stands in for a low day: the offer exists and is withheld
	// until something asks anyway.
	offer *squirrel.Offer
	gated bool
	// What the offer's buttons did, so a test can assert on the write rather
	// than on a rendering of it.
	answers    []string
	refused    []int64
	subscribed []string

	// What was written, so a test can assert on the write rather than on a
	// rendering of it. inserted is every note's words in the order they were
	// stored; states is where each id ended up.
	inserted []string
	states   map[int64]squirrel.ItemState

	// Fixed points created by a proposal that was pressed.
	moments []squirrel.Moment

	// Every press of "that landed badly", so a test can assert on the record
	// rather than on a rendering of it. noReplyToMark stands for a sheet with
	// nothing behind it to mark.
	landedBadly   []time.Time
	noReplyToMark bool

	// One fixed point, what is coming, and the notes pointing at one.
	moment   *squirrel.Moment
	upcoming []squirrel.Moment
	attached []squirrel.Item
	// What was pointed where, and what was put back.
	attachedTo    []int64
	attachedItems []int64
	detached      []int64

	// What was set aside, and what was picked back up.
	aside  []squirrel.HeldItem
	unheld []int64

	// The sequence in progress, newest first, and what was done to it.
	steps    []squirrel.Step
	stepItem *int64
	finished []int64
	cleared  int

	// The conversation: what has been said, what this render said, and
	// whether there is a page above.
	turns       []squirrel.Turn
	appended    []squirrel.Turn
	moreTurns   bool
	pagedBefore int64
	// The reading this render wrote, so a test can assert on the write rather
	// than on a rendering of it.
	recorded squirrel.Mood
	// What each door is holding, and a failure that belongs to the counting
	// alone — the doors have to survive it while the rest of the page works.
	waiting    squirrel.Waiting
	waitingErr error
	// A failure that belongs to reading the chores alone, so a test about the
	// door can fail that read while the conversation itself still renders.
	choresErr error

	// What the chore handlers did, so a test can assert on the write rather
	// than on a rendering of it.
	completed  []int64
	retired    []int64
	reinterval struct {
		name  string
		every time.Duration
	}
}

var errTest = errors.New("connection refused")

func (f *fakeStore) ActiveChores(_ context.Context, _ int64) ([]squirrel.Chore, error) {
	if f.choresErr != nil {
		return nil, f.choresErr
	}
	if f.err != nil {
		return nil, f.err
	}
	return f.chores, nil
}

func (f *fakeStore) SearchChores(_ context.Context, _ int64, q string, limit int) ([]squirrel.Chore, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := []squirrel.Chore{}
	for _, c := range f.chores {
		if c.Active && strings.Contains(strings.ToLower(c.Name), strings.ToLower(q)) {
			out = append(out, c)
		}
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (f *fakeStore) DeactivateChore(_ context.Context, choreID int64) error {
	if f.err != nil {
		return f.err
	}
	f.retired = append(f.retired, choreID)
	return nil
}

func (f *fakeStore) RecordCompletion(_ context.Context, choreID, _ int64, _ string, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.completed = append(f.completed, choreID)
	return nil
}

func (f *fakeStore) UpsertChore(_ context.Context, _ int64, name string, every, _ time.Duration) (squirrel.Chore, error) {
	if f.err != nil {
		return squirrel.Chore{}, f.err
	}
	f.reinterval.name, f.reinterval.every = name, every
	return squirrel.Chore{Name: name, Every: every}, nil
}

func (f *fakeStore) UpsertChoreAsking(_ context.Context, _ int64, name string, every, _ time.Duration, ask squirrel.Asking) (squirrel.Chore, error) {
	if f.err != nil {
		return squirrel.Chore{}, f.err
	}
	c := squirrel.Chore{
		ID: int64(len(f.chores) + 1), Name: name, Active: true,
		Every: every, EveryDays: int(every.Hours() / 24), Ask: ask,
	}
	f.chores = append(f.chores, c)
	return c, nil
}

// The timer. One per person, replaced each time — the fake keeps exactly what
// the store keeps, which is the current one and nothing else.
func (f *fakeStore) StartTimer(_ context.Context, _ int64, label string, d time.Duration, at time.Time) (squirrel.Timer, error) {
	if f.err != nil {
		return squirrel.Timer{}, f.err
	}
	t := squirrel.Timer{Label: label, Started: at, Ends: at.Add(d)}
	f.timer = &t
	return t, nil
}

func (f *fakeStore) CurrentTimer(_ context.Context, _ int64) (squirrel.Timer, bool, error) {
	if f.err != nil || f.timer == nil {
		return squirrel.Timer{}, false, f.err
	}
	return *f.timer, true, nil
}

func (f *fakeStore) StopTimer(_ context.Context, _ int64) error {
	if f.err != nil {
		return f.err
	}
	f.timer = nil
	return nil
}

func (f *fakeStore) OpenItems(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if it.State == squirrel.ItemOpen && it.Kind != squirrel.ItemTask {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

// OpenItemsAfter is the pile from a cursor. The fake keeps the store's rule
// that a cursor naming nothing is no cursor at all.
func (f *fakeStore) OpenItemsAfter(ctx context.Context, personID, afterID int64, limit int) ([]squirrel.Item, bool, error) {
	all, _, err := f.OpenItems(ctx, personID, len(f.items)+1)
	if err != nil || afterID == 0 {
		return f.OpenItems(ctx, personID, limit)
	}
	seen := false
	for _, it := range f.items {
		if it.ID == afterID {
			seen = true
		}
	}
	if !seen {
		return f.OpenItems(ctx, personID, limit)
	}
	out := []squirrel.Item{}
	past := false
	for _, it := range all {
		if past {
			out = append(out, it)
		}
		if it.ID == afterID {
			past = true
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

func (f *fakeStore) KeptItems(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if it.State == squirrel.ItemKept {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

// InsertItem is the slot. The fake keeps the store's own contract: a fresh row
// answers true, and the payload marks it a note by construction.
func (f *fakeStore) InsertItem(_ context.Context, i squirrel.Item) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	id := int64(len(f.items) + 1)
	f.items = append([]squirrel.Item{{
		ID: id, RawText: i.RawText, ReceivedAt: i.ReceivedAt, State: squirrel.ItemOpen,
	}}, f.items...)
	f.inserted = append(f.inserted, i.RawText)
	return true, nil
}

// The picker, faked. The rules themselves are proved against a real database
// in internal/squirrel; what the screen has to be tested for is what it does
// with an offer and with the absence of one, so this hands back whatever the
// test set and records the answers.
func (f *fakeStore) PickNow(_ context.Context, _ int64, _ time.Time, showAnyway bool) (squirrel.Offer, bool, error) {
	if f.err != nil {
		return squirrel.Offer{}, false, f.err
	}
	if f.offer == nil || (f.gated && !showAnyway) {
		return squirrel.Offer{}, false, nil
	}
	return *f.offer, true, nil
}

func (f *fakeStore) Did(_ context.Context, _ int64, o squirrel.Offer, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.answers = append(f.answers, string(squirrel.AnswerDid)+":"+string(o.Kind))
	return nil
}

func (f *fakeStore) Refuse(_ context.Context, _ int64, kind squirrel.OfferKind, refID int64, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.refused = append(f.refused, refID)
	f.answers = append(f.answers, string(squirrel.AnswerLater)+":"+string(kind))
	return nil
}

func (f *fakeStore) RecordAnswer(_ context.Context, _ int64, kind squirrel.OfferKind, _ int64, answer squirrel.OfferAnswer, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.answers = append(f.answers, string(answer)+":"+string(kind))
	return nil
}

func (f *fakeStore) SaveSubscription(_ context.Context, _ int64, sub squirrel.Subscription) error {
	if f.err != nil {
		return f.err
	}
	f.subscribed = append(f.subscribed, sub.Endpoint)
	return nil
}

func (f *fakeStore) RecordCheckin(_ context.Context, _ int64, m squirrel.Mood, _ string, at time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.checkin = &squirrel.Checkin{Mood: m, SaidAt: at}
	f.recorded = m
	return nil
}

func (f *fakeStore) CheckinsSince(_ context.Context, _ int64, since time.Time) ([]squirrel.Checkin, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := []squirrel.Checkin{}
	for _, c := range f.readings {
		if !c.SaidAt.Before(since) {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeStore) LatestCheckin(_ context.Context, _ int64) (squirrel.Checkin, bool, error) {
	if f.err != nil {
		return squirrel.Checkin{}, false, f.err
	}
	if f.checkin == nil {
		return squirrel.Checkin{}, false, nil
	}
	return *f.checkin, true, nil
}

func (f *fakeStore) Reword(_ context.Context, _ int64, id int64, text string) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].RawText = text
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) Tasks(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	return f.ofKind(squirrel.ItemTask, squirrel.ItemOpen, limit)
}

func (f *fakeStore) ArchivedTasks(_ context.Context, _ int64, limit int) ([]squirrel.Item, bool, error) {
	return f.ofKind(squirrel.ItemTask, squirrel.ItemDone, limit)
}

func (f *fakeStore) ofKind(k squirrel.ItemKind, st squirrel.ItemState, limit int) ([]squirrel.Item, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Item{}
	for _, it := range f.items {
		if it.Kind == k && it.State == st {
			out = append(out, it)
		}
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}

func (f *fakeStore) SetItemKind(_ context.Context, _ int64, id int64, k squirrel.ItemKind) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for i := range f.items {
		if f.items[i].ID == id {
			f.items[i].Kind = k
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) InsertItemReturningID(_ context.Context, i squirrel.Item) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	id := int64(len(f.items) + 1)
	f.items = append([]squirrel.Item{{
		ID: id, RawText: i.RawText, ReceivedAt: i.ReceivedAt,
		State: squirrel.ItemOpen, Kind: squirrel.ItemNote,
	}}, f.items...)
	return id, nil
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

// MoveItemState is the conditional write the deck uses, and the fake keeps the
// real one's rule: already being at the target is a success, because a second
// identical press is a press.
func (f *fakeStore) MoveItemState(_ context.Context, id int64, from, to squirrel.ItemState, _ time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for i := range f.items {
		if f.items[i].ID != id {
			continue
		}
		if f.items[i].State != from && f.items[i].State != to {
			return false, nil
		}
		f.items[i].State = to
		if f.states == nil {
			f.states = map[int64]squirrel.ItemState{}
		}
		f.states[id] = to
		return true, nil
	}
	return false, nil
}

// LandedBadlyLatest records that the last thing Buddy said did not land. The
// fake keeps the real one's shape: it answers whether there was anything to
// mark, so a press with nothing behind it is not reported as heard.
func (f *fakeStore) LandedBadlyLatest(_ context.Context, _ int64, at time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	if f.landedBadly == nil {
		f.landedBadly = []time.Time{}
	}
	if f.noReplyToMark {
		return false, nil
	}
	f.landedBadly = append(f.landedBadly, at)
	return true, nil
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
	if f.states == nil {
		f.states = map[int64]squirrel.ItemState{}
	}
	f.states[id] = state
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
		if wantMethod != method {
			continue
		}
		// `{$}` is Go's "this path and nothing under it". Without honouring it
		// here, "/" would answer for every URL on the screen and the home
		// screen's tests would pass for the wrong reason.
		if exact, ok := strings.CutSuffix(path, "{$}"); ok {
			asked, _, _ := strings.Cut(target, "?")
			if asked != exact {
				continue
			}
		} else if !strings.HasPrefix(target, path) {
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
	return squirrel.Item{
		ID: id, RawText: text, ReceivedAt: time.Date(2026, 8, 14, 9, 0, 0, 0, time.UTC),
		State: state, Kind: squirrel.ItemNote,
	}
}

// task is a note that was decided on. Same row, different kind — which is the
// whole model.
func task(id int64, text string, state squirrel.ItemState) squirrel.Item {
	it := note(id, text, state)
	it.Kind = squirrel.ItemTask
	return it
}

// Things you cannot act on, faked. The transitions themselves are proved
// against a real database in internal/squirrel; what the screen has to be
// tested for is what it draws and what it refuses to draw.
func (f *fakeStore) HoldItem(_ context.Context, _, itemID int64, state squirrel.ItemState, because string, _ time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for i := range f.items {
		if f.items[i].ID == itemID && f.items[i].State == squirrel.ItemOpen {
			f.items[i].State = state
			f.aside = append(f.aside, squirrel.HeldItem{
				ID: itemID, Text: f.items[i].RawText, State: state,
				Because: because, Kind: f.items[i].Kind,
			})
			return true, nil
		}
	}
	return false, nil
}

func (f *fakeStore) HeldItems(_ context.Context, _ int64, limit int) ([]squirrel.HeldItem, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	if len(f.aside) > limit {
		return f.aside[:limit], true, nil
	}
	return f.aside, false, nil
}

func (f *fakeStore) Unhold(_ context.Context, _, itemID int64, _ time.Time) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	for i, h := range f.aside {
		if h.ID == itemID {
			f.aside = append(f.aside[:i], f.aside[i+1:]...)
			f.unheld = append(f.unheld, itemID)
			return true, nil
		}
	}
	return false, nil
}

// A fixed point the coach proposed and you kept. The screen's own `!at`
// equivalent, faked; the moment parsing itself is proved in internal/squirrel.
func (f *fakeStore) CreateMoment(_ context.Context, _ int64, m squirrel.Moment) (squirrel.Moment, error) {
	if f.err != nil {
		return squirrel.Moment{}, f.err
	}
	f.moments = append(f.moments, m)
	return m, nil
}

// The step half of the fake store. One sequence, replaced wholesale, the same
// way the real one behaves — and, like the real one, with no way to ask it for
// the list.
func (f *fakeStore) SaveSteps(_ context.Context, _ int64, itemID *int64, label string, steps []string) error {
	if f.err != nil {
		return f.err
	}
	f.steps = nil
	for i, body := range steps {
		f.steps = append(f.steps, squirrel.Step{
			ID: int64(i + 1), Label: label, Body: body, Last: i == len(steps)-1,
		})
	}
	f.stepItem = itemID
	return nil
}

func (f *fakeStore) NextStep(_ context.Context, _ int64) (squirrel.Step, bool, error) {
	if f.err != nil {
		return squirrel.Step{}, false, f.err
	}
	if len(f.steps) == 0 {
		return squirrel.Step{}, false, nil
	}
	return f.steps[0], true, nil
}

func (f *fakeStore) StepDone(_ context.Context, _ int64, stepID int64, _ time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.finished = append(f.finished, stepID)
	if len(f.steps) > 0 && f.steps[0].ID == stepID {
		f.steps = f.steps[1:]
	}
	return nil
}

func (f *fakeStore) ClearSteps(_ context.Context, _ int64) error {
	if f.err != nil {
		return f.err
	}
	f.steps = nil
	f.cleared++
	return nil
}

// fakePhotos stands in for the volume. The durability — write, fsync, rename,
// fsync the directory — is proved in internal/squirrel; what the screen has to
// be tested for is that it offers a camera only when there is somewhere to put
// a photograph, and what it stores when there is.
type fakePhotos struct {
	kept []string
	err  error
}

func (p *fakePhotos) Keep(r io.Reader, contentType string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}
	p.kept = append(p.kept, string(b))
	return "photo-1.jpg", nil
}

func (p *fakePhotos) Open(string) (*os.File, error) { return nil, os.ErrNotExist }

// fakeSpool stands in for the durable half of capture. The spool's own
// durability — write, fsync, rename, fsync the directory — is proved in
// internal/squirrel; what the screen has to be tested for is that it goes
// through one at all, and what it does when it cannot.
type fakeSpool struct {
	written  []squirrel.Capture
	err      error
	readonly bool
}

func (s *fakeSpool) Write(c squirrel.Capture) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.written = append(s.written, c)
	return "spooled", nil
}

func (s *fakeSpool) Writable() bool { return !s.readonly }

// mountedSpooling is mounted with a spool the test can inspect, for the two
// things that are about capture itself rather than about the pile.
// mountedWithCamera is mounted plus somewhere to keep a photograph.
func mountedWithCamera(t *testing.T, f *fakeStore, sp *fakeSpool, ph *fakePhotos) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: sp, Photos: ph,
	}))
	return m
}

func mountedSpooling(t *testing.T, f *fakeStore, sp *fakeSpool) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: sp,
	}))
	return m
}

func mounted(t *testing.T, f *fakeStore) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{},
	}))
	return m
}

// fakeCoach stands in for the model, and its zero value is the shipping
// configuration with no key: nothing behind the box, and the four chips
// answering on their own.
type fakeCoach struct {
	reply string
	err   error
	// asked is every turn it was handed, so a test can assert on what the
	// model was told rather than on a rendering of it.
	asked []struct{ kind, said, subject string }
	// talk is the window, which the real one keeps in memory too.
	talk   []Exchange
	forgot int

	// decision is what the model chooses instead of the picker's answer, or
	// nil for "the picker was right". picked records what it was shown.
	decision *fakeDecision
	picked   []string
	// peeked counts the asks that were not allowed to pay, so a test can show
	// a surface consulted the cache rather than the model.
	peeked int

	// steps is what a breakdown returns, and empty means it could not do one.
	// broke counts the asks, so a test can show the three blockers that are
	// not "too big" never reach it.
	steps        []string
	broke        int
	brokeDown    string
	brokeBlocker string

	// pieces is what a split proposes, and empty means it could not. splittable
	// is the free rule that decides whether the press is drawn at all.
	pieces     []string
	splittable bool

	// did is what a turn actually changed, and propose is what it wants
	// permission for. Both empty on an ordinary turn, which is most of them.
	did     []string
	propose *Proposal

	// spent and ceiling are what the sheet reports. Empty spent stands in for
	// a sum that could not be read, which must draw no line at all.
	spent   string
	ceiling string
}

type fakeDecision struct {
	kind    string
	refID   int64
	text    string
	because string
}

func (c *fakeCoach) ask(_ context.Context, _ int64, kind, said, subject string) (Answer, error) {
	c.asked = append(c.asked, struct{ kind, said, subject string }{kind, said, subject})
	if c.err != nil {
		return Answer{}, c.err
	}
	return Answer{Text: c.reply, Did: c.did, Propose: c.propose}, nil
}

// decided is what the model chooses instead, when a test says it chooses
// anything. The zero value chooses nothing, which is the shipping state
// whenever the picker's answer is good enough or nothing is configured.
func (c *fakeCoach) decide(_ context.Context, _ int64, pickedKind string, pickedRef int64,
	mayAsk bool) (string, int64, string, string, bool) {

	c.picked = append(c.picked, pickedKind)
	if !mayAsk {
		// Stands in for a cold cache, which is the case that matters: a
		// surface that may not pay has nothing to show and must fall back.
		c.peeked++
		return "", 0, "", "", false
	}
	if c.decision == nil {
		return "", 0, "", "", false
	}
	d := *c.decision
	return d.kind, d.refID, d.text, d.because, true
}

func (c *fakeCoach) smaller(_ context.Context, _ int64, task, blocker string) ([]string, bool) {
	c.broke++
	c.brokeDown, c.brokeBlocker = task, blocker
	return c.steps, len(c.steps) > 0
}

func (c *fakeCoach) options(o Options) Options {
	o.Ask = c.ask
	o.Decide = c.decide
	o.Smaller = c.smaller
	o.Split = func(_ context.Context, _ int64, _ string) ([]string, bool) {
		return c.pieces, len(c.pieces) > 0
	}
	o.Splittable = func(string) bool { return c.splittable }
	o.Spent = func(context.Context, int64) (string, string, bool) {
		return c.spent, c.ceiling, c.spent != ""
	}
	o.Recent = func(int64) []Exchange { return c.talk }
	o.Remember = func(_ int64, said, replied string) {
		c.talk = append(c.talk, Exchange{Said: said, Replied: replied})
	}
	o.Forget = func(int64) { c.forgot++; c.talk = nil }
	return o
}

// mountedWith is mounted plus a coach. Passing nil mounts the screen with no
// coach at all, which is what ships whenever no key is set and therefore what
// most of these tests should still work under.
func mountedWith(t *testing.T, f *fakeStore, c *fakeCoach) *testMux {
	t.Helper()
	m := newTestMux()
	opts := Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 }, Spool: &fakeSpool{},
	}
	if c != nil {
		opts = c.options(opts)
	}
	require.NoError(t, Mount(m, f, opts))
	return m
}

// One fixed point and what is pointing at it. The fields are per-write like
// every other fake here, so a test asserts on what was written rather than on
// a rendering of it.
func withMoment(m *squirrel.Moment) *fakeStore { return &fakeStore{moment: m} }

func (f *fakeStore) MomentByID(_ context.Context, _, id int64) (squirrel.Moment, bool, error) {
	if f.err != nil {
		return squirrel.Moment{}, false, f.err
	}
	if f.moment == nil || f.moment.ID != id {
		return squirrel.Moment{}, false, nil
	}
	return *f.moment, true, nil
}

func (f *fakeStore) Upcoming(_ context.Context, _ int64, _ time.Time, _ int) ([]squirrel.Moment, error) {
	return f.upcoming, f.err
}

func (f *fakeStore) NotesFor(_ context.Context, _, _ int64) ([]squirrel.Item, error) {
	return f.attached, f.err
}

func (f *fakeStore) AttachNote(_ context.Context, _, itemID, momentID int64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.attachedTo = append(f.attachedTo, momentID)
	f.attachedItems = append(f.attachedItems, itemID)
	return true, nil
}

func (f *fakeStore) DetachNote(_ context.Context, _, itemID int64) (bool, error) {
	if f.err != nil {
		return false, f.err
	}
	f.detached = append(f.detached, itemID)
	return true, nil
}

// The conversation. turns is what has been said already; appended is what the
// handler under test said, so a test can assert on the write rather than on a
// rendering of it.
func (f *fakeStore) AppendTurn(_ context.Context, _ int64, t squirrel.Turn) (squirrel.Turn, error) {
	if f.err != nil {
		return squirrel.Turn{}, f.err
	}
	t.ID = int64(len(f.turns) + len(f.appended) + 1)
	t.SaidAt = now()
	f.appended = append(f.appended, t)
	return t, nil
}

func (f *fakeStore) RecentTurns(_ context.Context, _ int64, limit int) ([]squirrel.Turn, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	if len(f.turns) > limit {
		return f.turns[len(f.turns)-limit:], true, nil
	}
	return f.turns, f.moreTurns, nil
}

func (f *fakeStore) TurnsBefore(_ context.Context, _ int64, before int64, limit int) ([]squirrel.Turn, bool, error) {
	if f.err != nil {
		return nil, false, f.err
	}
	out := []squirrel.Turn{}
	for _, t := range f.turns {
		if t.ID < before {
			out = append(out, t)
		}
	}
	f.pagedBefore = before
	if len(out) > limit {
		return out[len(out)-limit:], true, nil
	}
	return out, false, nil
}

func (f *fakeStore) Waiting(_ context.Context, _ int64, _ time.Time) (squirrel.Waiting, error) {
	if f.waitingErr != nil {
		return squirrel.Waiting{}, f.waitingErr
	}
	if f.err != nil {
		return squirrel.Waiting{}, f.err
	}
	return f.waiting, nil
}

// opened presses a door and renders the conversation it produced.
//
// Two steps rather than one, because that is what actually happens: the press
// writes the turns, and the next render draws them. The fake keeps what was
// written in `appended`, so moving it into `turns` is what a real store would
// have done by the time the page came back.
func opened(t *testing.T, f *fakeStore, where string) string {
	t.Helper()
	// A fresh reading, so Buddy does not ask how you are on the way past and
	// become the live edge himself — which would take the controls off the
	// cards the door just put there. Incidental to what these tests are about,
	// and TestOnlyTheNewestBuddyTurnHasControls is where that rule is proved.
	if f.checkin == nil {
		f.checkin = &squirrel.Checkin{Mood: squirrel.MoodGood, SaidAt: now()}
	}
	m := routed(t, f)
	m.call(t, "POST", "/open", strings.NewReader("where="+where))
	f.turns, f.appended = append(f.turns, f.appended...), nil
	return m.call(t, "GET", "/", nil).Body.String()
}
