//go:build integration

package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Two people, two piles, and one assertion.
//
// This is the task worth doing even if the rest of the OIDC work were
// abandoned. Every store function already takes a personID and every one of
// them is scoped by it — but "already scoped" is exactly the claim that was
// never tested, because for the product's whole life there was one person and
// nothing to leak across.
//
// The shape is deliberate: it is a table of reads rather than a test per
// surface, so a new screen that forgets to scope is a line somebody has to
// deliberately not add.

// notMine is in every piece of the second person's pile, and in none of the
// first person's. It is one word so a body can be searched for it, and an
// unlikely one so a match is never a coincidence.
const notMine = "NOTMINEXYZZY"

// realStore opens Postgres for a test in package web. internal/squirrel has
// its own helper; this one exists because the sweep has to mount the actual
// screen against the actual store rather than against a fake that cannot leak.
func realStore(t *testing.T) *squirrel.Store {
	t.Helper()
	raw := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, raw, "TEST_DATABASE_URL is required — see docs/testing.md")

	ctx := context.Background()
	store, err := squirrel.OpenStore(ctx, raw)
	require.NoError(t, err)
	t.Cleanup(store.Close)
	require.NoError(t, store.Migrate(ctx))
	_, err = store.Pool().Exec(ctx,
		`truncate table prompt_lines, prompts, events, items, chores, identities, people
		 restart identity cascade`)
	require.NoError(t, err)
	return store
}

// aWholePile gives one person one of everything the screen can read.
func aWholePile(t *testing.T, store *squirrel.Store, handle, sub, mark string) int64 {
	t.Helper()
	ctx := context.Background()
	now := time.Now()

	personID, err := store.PersonForLogin(ctx, sub, handle)
	require.NoError(t, err)

	item := func(text string, state squirrel.ItemState, kind squirrel.ItemKind) int64 {
		id, err := store.InsertItemReturningID(ctx, squirrel.Item{
			Transport: squirrel.ScreenTransport, SenderID: &sub,
			PersonID: &personID, RawText: text, ReceivedAt: now,
			Payload: []byte(`{}`),
		})
		require.NoError(t, err)
		if kind != squirrel.ItemNote {
			_, err = store.SetItemKind(ctx, personID, id, kind)
			require.NoError(t, err)
		}
		if state != squirrel.ItemOpen {
			require.NoError(t, store.SetItemState(ctx, id, state, now))
		}
		return id
	}

	open := item(mark+" a note in the pile", squirrel.ItemOpen, squirrel.ItemNote)
	item(mark+" a note that was kept", squirrel.ItemKept, squirrel.ItemNote)
	item(mark+" a note that was dropped", squirrel.ItemDropped, squirrel.ItemNote)
	item(mark+" a task to do", squirrel.ItemOpen, squirrel.ItemTask)
	item(mark+" a task already done", squirrel.ItemDone, squirrel.ItemTask)

	// A photograph, so /photo/{id} and /photo/{id}/thumb have something of
	// somebody else's to refuse.
	_, err = store.InsertItemReturningID(ctx, squirrel.Item{
		Transport: squirrel.ScreenTransport, SenderID: &sub, PersonID: &personID,
		RawText: mark + " a photograph", ReceivedAt: now, Payload: []byte(`{}`),
		PhotoName: "photo-" + mark + ".jpg", PhotoType: "image/jpeg",
	})
	require.NoError(t, err)

	held := item(mark+" a note set aside", squirrel.ItemOpen, squirrel.ItemNote)
	_, err = store.HoldItem(ctx, personID, held, squirrel.ItemWaiting, mark+" waiting on the plumber", now)
	require.NoError(t, err)

	_, err = store.UpsertChore(ctx, personID, mark+" a chore", 7*24*time.Hour, 24*time.Hour)
	require.NoError(t, err)

	moment, err := store.CreateMoment(ctx, personID, squirrel.Moment{
		Label: mark + " an appointment", Starts: now.Add(3 * time.Hour),
		Travel: 20 * time.Minute, Ready: 10 * time.Minute, Bring: mark + " the letter",
	})
	require.NoError(t, err)
	_, err = store.AttachNote(ctx, personID, open, moment.ID)
	require.NoError(t, err)

	for _, turn := range []squirrel.Turn{
		{Who: squirrel.SpeakerYou, Words: mark + " something I said"},
		{Who: squirrel.SpeakerBuddy, Words: mark + " something Buddy said"},
	} {
		_, err = store.AppendTurn(ctx, personID, "buddy", turn)
		require.NoError(t, err)
	}

	require.NoError(t, store.RecordCheckin(ctx, personID, squirrel.MoodLow, "screen", now))
	require.NoError(t, store.SaveSteps(ctx, personID, nil, mark+" a thing in steps",
		[]string{mark + " the first step", mark + " the second step"}))
	// A run in progress. It carries no text of its own, so the marker cannot
	// leak through it — what would leak is being offered somebody else's place
	// back, which TestARunBelongsToOnePerson pins at the store.
	require.NoError(t, store.MarkRun(ctx, personID, squirrel.RunPile, now))

	return personID
}

// theSweep mounts the real screen against the real store, signed in as the
// first person, and asks it everything it can be asked.
func theSweep(t *testing.T) (*realMux, *squirrel.Store, int64, int64) {
	t.Helper()
	store := realStore(t)

	mine := aWholePile(t, store, "mine", "sub-mine", "MINE")
	theirs := aWholePile(t, store, "theirs", "sub-theirs", notMine)

	// A real ServeMux, not the test one: the sweep aims reads at rows by id
	// and the test mux matches by prefix, so it cannot resolve a {id}
	// wildcard.
	m := &realMux{mux: http.NewServeMux()}
	opts := signedInOptions()
	opts.Sessions = newSessions(signedInAs{personID: mine, sub: "sub-mine"}, cacheFor, cacheMost)
	opts.Photos = &fakePhotos{}
	opts.Location = time.UTC
	require.NoError(t, Mount(m, store, opts))
	return m, store, mine, theirs
}

// Every read the screen can perform, walked as one person, never showing the
// other one's pile.
func TestNoScreenShowsSomebodyElsesPile(t *testing.T) {
	m, store, mine, theirs := theSweep(t)
	ctx := context.Background()

	// Their rows, so a read can be aimed at one by id rather than only hoped
	// to stumble on it.
	theirItems, _, err := store.OpenItems(ctx, theirs, 50)
	require.NoError(t, err)
	require.NotEmpty(t, theirItems)
	theirNote := strconv.FormatInt(theirItems[0].ID, 10)

	theirMoments, err := store.Upcoming(ctx, theirs, time.Now(), 10)
	require.NoError(t, err)
	require.NotEmpty(t, theirMoments)
	theirMoment := strconv.FormatInt(theirMoments[0].ID, 10)

	theirChores, err := store.ActiveChores(ctx, theirs)
	require.NoError(t, err)
	require.NotEmpty(t, theirChores)
	theirChore := strconv.FormatInt(theirChores[0].ID, 10)

	// One line per surface. A screen added without a line here is a screen
	// nobody proved is scoped, which is the whole point of the shape.
	for _, read := range []struct {
		what   string
		method string
		target string
		form   url.Values
	}{
		{"the conversation", "GET", "/", nil},
		{"the pile's door", "POST", "/open", url.Values{"where": {"pile"}}},
		{"the kept door", "POST", "/open", url.Values{"where": {"kept"}}},
		{"the tasks door", "POST", "/open", url.Values{"where": {"tasks"}}},
		{"the chores door", "POST", "/open", url.Values{"where": {"chores"}}},
		{"what is coming", "POST", "/open", url.Values{"where": {"at"}}},
		{"what was set aside", "POST", "/open", url.Values{"where": {"held"}}},
		{"searching for their words", "POST", "/find", url.Values{"q": {notMine}}},
		{"searching for their chore", "POST", "/find", url.Values{"q": {"chore"}}},
		{"asking to search", "POST", "/find/ask", nil},
		{"their note, opened from a result", "POST", "/find/open",
			url.Values{"id": {theirNote}}},
		{"what Squirrel knows", "POST", "/knowing", nil},
		{"the readings", "GET", "/moods", nil},
		{"asking Buddy", "POST", "/buddy/ask", url.Values{"said": {"what should I do"}}},
		{"their note, opened", "POST", "/pile/why", url.Values{"id": {theirNote}}},
		{"their note, reworded", "POST", "/pile/reword", url.Values{"id": {theirNote}}},
		{"their note, asked about", "POST", "/pile/often", url.Values{"id": {theirNote}}},
		{"their note, acted on", "POST", "/pile/act", url.Values{"id": {theirNote}, "act": {"done"}}},
		{"their note, skipped", "POST", "/pile/later", url.Values{"id": {theirNote}}},
		{"their note, split", "POST", "/pile/split", url.Values{"id": {theirNote}}},
		{"their appointment", "GET", "/at/" + theirMoment, nil},
		{"their appointment, opened", "POST", "/at/open", url.Values{"id": {theirMoment}}},
		{"their appointment, noted", "POST", "/at/" + theirMoment + "/note",
			url.Values{"text": {"a note of mine"}}},
		{"their chore, acted on", "POST", "/chores/act",
			url.Values{"id": {theirChore}, "act": {"did"}}},
		{"their note, set aside", "POST", "/held/act",
			url.Values{"id": {theirNote}, "act": {"back"}}},
		{"their task, acted on", "POST", "/tasks/act",
			url.Values{"id": {theirNote}, "act": {"done"}}},
		{"the one thing", "POST", "/now/act", url.Values{"act": {"later"}}},
		{"a step", "POST", "/steps", url.Values{"act": {"done"}}},
		{"starting fresh", "POST", "/place/fresh", nil},
		{"their parked note, kept waiting", "POST", "/held/act",
			url.Values{"id": {theirNote}, "act": {"still"}}},
		{"a timer", "POST", "/timer", url.Values{"minutes": {"5"}}},
	} {
		var body string
		if read.form == nil {
			body = m.call(t, read.method, read.target, nil).Body.String()
		} else {
			body = m.call(t, read.method, read.target,
				strings.NewReader(read.form.Encode())).Body.String()
		}
		require.NotContains(t, body, notMine,
			"%s showed somebody else's pile", read.what)
	}

	// And nothing that was written into the conversation along the way is
	// theirs either. A turn is the one thing on this screen that outlives the
	// response it was rendered in: a leak that reached a turn is a leak that
	// is still on the screen tomorrow.
	turns, _, err := store.RecentTurns(ctx, mine, "buddy", 200)
	require.NoError(t, err)
	for _, turn := range turns {
		// Shown is the rendered pile — cards, chores, appointments — and none
		// of it may be theirs whoever spoke.
		require.NotContains(t, string(turn.Shown), notMine, "a turn carries their pile")
		if turn.Who == squirrel.SpeakerYou {
			// My own words are mine by definition, and the sweep types the
			// marker itself into the search box. What that echo proves is that
			// searching for somebody else's words finds nothing, which the
			// table above already asserted on the response.
			continue
		}
		require.NotContains(t, turn.Words, notMine, "Buddy said their words back")
	}

	// Nor did any of it write into their conversation.
	theirTurns, _, err := store.RecentTurns(ctx, theirs, "buddy", 200)
	require.NoError(t, err)
	require.Len(t, theirTurns, 2, "somebody else's conversation was written to")
}

// A photograph belonging to somebody else is not found, rather than refused.
// 404 and not 403: telling a stranger that a row exists is telling them
// something about a pile that is not theirs.
func TestSomebodyElsesPhotographIsNotFound(t *testing.T) {
	_, store, _, theirs := theSweep(t)
	ctx := context.Background()

	var photoID int64
	require.NoError(t, store.Pool().QueryRow(ctx,
		`select id from items where person_id = $1 and attachment_path is not null limit 1`,
		theirs).Scan(&photoID))

	opts := signedInOptions()
	opts.Photos = &fakePhotos{}
	id := strconv.FormatInt(photoID, 10)

	// Both routes. They were the same string twice here, so the card-sized
	// copy — the one every card on the screen actually asks for — was never
	// aimed at somebody else's row at all.
	for _, serves := range []struct {
		what    string
		handler http.HandlerFunc
	}{
		{"/photo/{id}", photoHandler(store, opts)},
		{"/photo/{id}/thumb", thumbHandler(store, opts)},
	} {
		r := httptest.NewRequest("GET", "/photo/"+id, nil)
		r.SetPathValue("id", id)
		w := httptest.NewRecorder()
		serves.handler(w, withWho(r, 1, "sub-mine"))
		require.Equal(t, http.StatusNotFound, w.Code, "%s served it", serves.what)
	}
}
