package web

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func chore(id int64, name string, everyDays, sinceDays int) squirrel.Chore {
	return squirrel.Chore{
		ID: id, PersonID: 1, Name: name, Active: true, EverDone: true,
		Every:     time.Duration(everyDays) * 24 * time.Hour,
		EveryDays: everyDays,
		SinceDays: sinceDays,
	}
}

func TestChoresListsWhatComesBack(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 14, 3),
		chore(2, "water the plants", 7, 1),
	}}
	body := mounted(t, f).call(t, "GET", "/chores", nil).Body.String()

	require.Contains(t, body, "bins out")
	require.Contains(t, body, "water the plants")
	require.Contains(t, body, "EVERY 2 WEEKS", "the rhythm as a person says it, not arithmetic")
}

// The rule that governs the pile governs this too: a chore may say when it was
// last done, never how many are waiting or how far behind anything is.
func TestChoresNeverCounts(t *testing.T) {
	chores := []squirrel.Chore{}
	for i := int64(1); i <= 7; i++ {
		chores = append(chores, chore(i, "chore "+string(rune('a'+i)), 7, 30))
	}
	body := mounted(t, &fakeStore{chores: chores}).call(t, "GET", "/chores", nil).Body.String()

	lower := strings.ToLower(body)
	for _, forbidden := range []string{"7 chores", "overdue", "behind", "streak", "% ", "of 7"} {
		require.NotContains(t, lower, forbidden)
	}
}

func TestTheEmptyChoreListDoesNotNag(t *testing.T) {
	body := mounted(t, &fakeStore{}).call(t, "GET", "/chores", nil).Body.String()

	require.Contains(t, body, "nothing comes back")
	for _, forbidden := range []string{"should", "why not", "add one"} {
		require.NotContains(t, strings.ToLower(body), forbidden)
	}
}

func TestMarkingAChoreDoneRecordsIt(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 20)}}
	w := post(t, mounted(t, f), "/chores/act", url.Values{"id": {"1"}, "act": {"done"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, []int64{1}, f.completed)
}

func TestRetiringAChoreFromTheScreen(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/chores/act", url.Values{"id": {"1"}, "act": {"retire"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, []int64{1}, f.retired)
}

// The same interval vocabulary the card uses, so the two surfaces cannot drift
// into meaning different things by "every 2 weeks".
func TestChangingHowOftenAChoreComesBack(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/chores/act",
		url.Values{"id": {"1"}, "every": {"every week"}})

	require.Equal(t, 303, w.Code)
	require.Equal(t, "bins out", f.reinterval.name)
	require.Equal(t, 7*24*time.Hour, f.reinterval.every)
}

// An id from a form is a number a stranger could have typed, so it is checked
// against what this person actually has rather than trusted.
func TestActingOnAChoreThatIsNotYours(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/chores/act", url.Values{"id": {"99"}, "act": {"retire"}})

	require.Equal(t, 404, w.Code)
	require.Empty(t, f.retired)
}

func TestChoresRefusesAnUnknownAction(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}
	w := post(t, mounted(t, f), "/chores/act", url.Values{"id": {"1"}, "act": {"delete"}})

	require.Equal(t, 400, w.Code)
	require.Empty(t, f.retired)
}

// The two screens have to be reachable from each other, or the chores are as
// invisible as they were before.
// With three screens the lid carries both of the others: one link that cycled
// would put the chores two presses from the pile.
func TestTheLidOffersTheTwoPlacesYouAreNot(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	for _, c := range []struct {
		on    string
		wants []string
		not   string
	}{
		{"/pile", []string{`href="/tasks"`, `href="/chores"`}, `class="lidlink" href="/pile"`},
		{"/tasks", []string{`href="/pile"`, `href="/chores"`}, `class="lidlink" href="/tasks"`},
		{"/chores", []string{`href="/pile"`, `href="/tasks"`}, `class="lidlink" href="/chores"`},
		// The archive belongs to the tasks and the shelf to the pile: each is
		// reached from one of them, and that is what you would be looking for
		// the way back to.
		{"/tasks/done", []string{`href="/pile"`, `href="/chores"`}, `class="lidlink" href="/tasks"`},
		{"/kept", []string{`href="/tasks"`, `href="/chores"`}, `class="lidlink" href="/pile"`},
	} {
		body := m.call(t, "GET", c.on, nil).Body.String()
		for _, want := range c.wants {
			require.Contains(t, body, want, c.on)
		}
		require.NotContains(t, body, c.not, "a link to where you already are is furniture")
	}
}

// A chore nobody has ever done has a baseline anyway — its own birthday — and
// reporting that as "last done" would be a sentence about the person.
func TestAChoreNeverDoneSaysOnlyItsRhythm(t *testing.T) {
	never := chore(1, "descale the shower head", 30, 400)
	never.EverDone = false
	body := mounted(t, &fakeStore{chores: []squirrel.Chore{never}}).
		call(t, "GET", "/chores", nil).Body.String()

	require.Contains(t, body, "EVERY MONTH")
	require.NotContains(t, body, "LAST DONE")
	require.NotContains(t, body, "a while back")
}

// The chore is at rest here, so it does not wear the colour of something being
// made, nor the page tab that says what a note ended up as.
func TestAChoreAtRestIsNotDressedAsANoteOrACreation(t *testing.T) {
	body := mounted(t, &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}).
		call(t, "GET", "/chores", nil).Body.String()

	require.NotContains(t, body, "state-chore")
	require.NotContains(t, body, `class="rcard`)
	require.NotContains(t, body, `class="tab"`)
}

func TestChoresFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{err: errTest}
	w := mounted(t, f).call(t, "GET", "/chores", nil)

	require.Equal(t, 503, w.Code)
	require.Contains(t, w.Body.String(), "cannot reach")
}

// Roughly when, never how long. An exact day count on a chore you have not
// done is a number that grows while you are not looking, which is the thing
// this product does not do — and "3 days" is one small step from "3 days
// late".
func TestAChoreSaysRoughlyWhenNotHowLong(t *testing.T) {
	for _, tc := range []struct {
		since int
		want  string
	}{
		{0, "today"},
		{1, "yesterday"},
		{3, "this week"},
		{6, "this week"},
		{9, "last week"},
		{13, "last week"},
		{20, "this month"},
		{45, "a while back"},
		{400, "a while back"},
	} {
		require.Equal(t, tc.want, lastDone(tc.since), "%d days", tc.since)
	}
}

func TestTheChoresScreenNeverPrintsADayCount(t *testing.T) {
	body := mounted(t, &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 14, 3),
		chore(2, "water the plants", 7, 45),
	}}).call(t, "GET", "/chores", nil).Body.String()

	require.Contains(t, body, "this week")
	require.Contains(t, body, "a while back")
	require.NotContains(t, body, "3 days ago")
	require.NotContains(t, body, "45 days")
}

// aChore is one that exists, so the handler's own id check passes.
func aChore() []squirrel.Chore {
	return []squirrel.Chore{{
		ID: 1, Name: "water the plants", Every: 7 * 24 * time.Hour,
		EveryDays: 7, SinceDays: 8, Active: true, EverDone: true,
	}}
}

// Doing a chore says so, and the saying is in the record beside the doing.
func TestDoingAChoreIsSaid(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=done"))

	require.Equal(t, []int64{1}, f.completed)
	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "water the plants")
}

// The words come from the stored chore rather than from the form, so what the
// record says happened cannot be something the press claimed.
func TestWhatIsSaidComesFromTheStoredChore(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&act=done&label=something+else+entirely"))

	require.Contains(t, f.appended[0].Words, "water the plants")
	require.NotContains(t, f.appended[0].Words, "something else entirely")
}

// Stopping one is not the same sentence as doing it — which answer you gave is
// the whole of what happened.
func TestRetiringAChoreSaysSomethingElse(t *testing.T) {
	did, stopped := &fakeStore{chores: aChore()}, &fakeStore{chores: aChore()}
	routed(t, did).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=done"))
	routed(t, stopped).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=retire"))

	require.Equal(t, []int64{1}, stopped.retired)
	require.NotEqual(t, did.appended[1].Words, stopped.appended[1].Words)
}

// An act nobody offered does nothing and says nothing.
func TestAChoreActThatWasNeverOfferedDoesNothing(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act", strings.NewReader("id=1&act=burn"))

	require.Empty(t, f.appended)
	require.Empty(t, f.completed)
	require.Empty(t, f.retired)
}

// A new chore comes back as a card, so the thing you just made is on the screen
// rather than somewhere you have to go and look.
func TestANewChoreComesBackAsACard(t *testing.T) {
	f := &fakeStore{}
	routed(t, f).call(t, "POST", "/chores/new",
		strings.NewReader("name=descale+the+kettle&every=every+2+weeks"))

	require.Len(t, f.chores, 1)
	require.Equal(t, "descale the kettle", f.chores[0].Name)
	require.Len(t, f.appended, 2)
	require.Contains(t, string(f.appended[1].Shown), "descale the kettle")
	require.Contains(t, string(f.appended[1].Shown), "/chores/act")
}
