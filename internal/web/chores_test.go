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
	body := opened(t, f, "chores")

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
	body := opened(t, &fakeStore{chores: chores}, "chores")

	lower := strings.ToLower(body)
	for _, forbidden := range []string{"7 chores", "overdue", "behind", "streak", "% ", "of 7"} {
		require.NotContains(t, lower, forbidden)
	}
}

func TestTheEmptyChoreListDoesNotNag(t *testing.T) {
	body := opened(t, &fakeStore{}, "chores")

	require.Contains(t, strings.ToLower(body), "nothing comes back")
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
//
// Two places since 24 August 2026, not three: the tasks and the chores stopped
// being pages and became messages, and the way to them is the rail on the one
// screen you reach them from.
func TestTheLidOffersTheOtherPlaceYouAreNot(t *testing.T) {
	f := &fakeStore{items: []squirrel.Item{note(1, "buy milk", squirrel.ItemOpen)}}
	m := mounted(t, f)

	for _, c := range []struct {
		on    string
		wants []string
		not   string
	}{
		{"/pile", []string{`href="/at"`}, `class="lidlink" href="/pile"`},
		{"/at", []string{`href="/pile"`}, `class="lidlink" href="/at"`},
		// The shelf belongs to the pile: it is reached from there, and that is
		// what you would be looking for the way back to.
		{"/kept", []string{`href="/at"`}, `class="lidlink" href="/pile"`},
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
	body := opened(t, &fakeStore{chores: []squirrel.Chore{never}}, "chores")

	require.Contains(t, body, "EVERY MONTH")
	require.NotContains(t, body, "LAST DONE")
	require.NotContains(t, body, "a while back")
}

// The chore is at rest here, so it does not wear the colour of something being
// made, nor the page tab that says what a note ended up as.
func TestAChoreAtRestIsNotDressedAsANoteOrACreation(t *testing.T) {
	body := opened(t, &fakeStore{chores: []squirrel.Chore{chore(1, "bins out", 14, 3)}}, "chores")

	require.NotContains(t, body, "state-chore")
	require.NotContains(t, body, `class="rcard`)
	require.NotContains(t, body, `class="tab"`)
}

func TestChoresFailsVisiblyWhenTheDatabaseIsDown(t *testing.T) {
	f := &fakeStore{choresErr: errTest}
	// The door still opens and Buddy says he cannot reach them. A 503 was
	// right while this was a page of its own; the conversation cannot answer a
	// press with a status code, and an empty reply reads as a press that did
	// not land.
	body := opened(t, f, "chores")

	require.Contains(t, body, "cannot reach the chores")
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
	body := opened(t, &fakeStore{chores: []squirrel.Chore{
		chore(1, "bins out", 14, 3),
		chore(2, "water the plants", 7, 45),
	}}, "chores")

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

// Asking how often puts the question on the table with both rows on it.
func TestAskingHowOftenOffersNumbersAndUnits(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/often", strings.NewReader("id=1"))

	require.Len(t, f.appended, 2)
	shown := string(f.appended[1].Shown)
	for _, want := range []string{`"1"`, `"2"`, `"3"`, `"4"`, "days", "weeks", "months"} {
		require.Contains(t, shown, want)
	}
}

// The rhythm it has now is marked, so the question is answerable rather than a
// blank form: you are changing something, not inventing it.
func TestTheQuestionMarksTheRhythmItHasNow(t *testing.T) {
	f := &fakeStore{chores: []squirrel.Chore{{
		ID: 1, Name: "water the plants", Every: 14 * 24 * time.Hour,
		EveryDays: 14, Active: true,
	}}}
	routed(t, f).call(t, "POST", "/chores/often", strings.NewReader("id=1"))

	shown := string(f.appended[1].Shown)
	require.Contains(t, shown, `"chosen":"2"`)
	require.Contains(t, shown, `"chosen":"weeks"`)
}

// A rhythm the picker cannot say leaves both rows unmarked rather than rounding
// to the nearest offered thing. Marking the wrong one would say the chore is
// something it is not.
func TestARhythmThePickerCannotSayIsLeftUnmarked(t *testing.T) {
	// Ten days: not an offered number, not a whole number of weeks, not a
	// whole number of months. A quarter would not do — that is three months,
	// which the picker can say perfectly well.
	f := &fakeStore{chores: []squirrel.Chore{{
		ID: 1, Name: "the gutters", Every: 10 * 24 * time.Hour,
		EveryDays: 10, Active: true,
	}}}
	routed(t, f).call(t, "POST", "/chores/often", strings.NewReader("id=1"))

	require.NotContains(t, string(f.appended[1].Shown), `"chosen"`)
}

// Answering composes the same sentence the fixed chips used to post, so the
// chore path underneath is untouched.
func TestAnsweringHowOftenComposesTheSentence(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&count=3&unit=weeks"))

	require.Equal(t, 21*24*time.Hour, f.reinterval.every)
}

// And says it back in the same words a person would use.
func TestAnsweringHowOftenIsSaid(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&count=3&unit=weeks"))

	require.Len(t, f.appended, 2)
	require.Contains(t, f.appended[0].Words, "every 3 weeks")
}

// A number and a unit nobody offered do nothing. They arrive from a form.
func TestARhythmThatWasNeverOfferedDoesNothing(t *testing.T) {
	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&count=99&unit=fortnights"))

	require.Empty(t, f.appended)
	require.Zero(t, f.reinterval.every)
}

// The picker and a typed sentence produce the same interval for the same
// rhythm. Asserted on the duration, not on the string — this is what stops a
// later author replacing ParseEvery with arithmetic and redefining a month.
func TestThePickerAndTheSentenceAgree(t *testing.T) {
	_, typed, ok := squirrel.ParseEvery("every 3 months: water the plants")
	require.True(t, ok)

	f := &fakeStore{chores: aChore()}
	routed(t, f).call(t, "POST", "/chores/act",
		strings.NewReader("id=1&count=3&unit=months"))

	require.Equal(t, typed, f.reinterval.every)
}
