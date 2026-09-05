// Package coach is the one place a model is allowed to speak.
//
// On screen it is called Buddy, and the line is about authorship: anything a
// model wrote is Buddy's, anything the rules produced is Squirrel's. The
// picker's clause, the ladder's fixed sentences, the nudge that fires because a
// chore is due — those are Squirrel.
//
// Deliberately small and deliberately optional. Everything here has a
// deterministic answer underneath it, and the zero value is NoCoach, which
// answers "not available" to everything: with no key, no network, or a spent
// budget the product works as it did before this package existed. Every caller
// must be written so that is true.
//
// Nothing here imports internal/squirrel and internal/squirrel does not import
// this; internal/boot wires them, so the core cannot grow a dependency on a
// model being reachable.
package coach

import (
	"context"
	"errors"
	"time"
)

// ErrUnavailable is what every method returns when there is no coach: no key, no
// network, the budget spent, or an answer that failed its shape check. One error
// rather than a taxonomy, because the caller's only question is "did I get
// something usable".
var ErrUnavailable = errors.New("no coach available")

// And these say which, for the log and for nobody else. A retired model id reads
// exactly like a network blip otherwise, and the two want opposite things done
// about them.
var (
	// ErrProviderRefused is the provider answering, and saying no: a status
	// that is not 200. A retired model, a revoked key, a rate limit.
	ErrProviderRefused = errors.New("the provider refused")
	// ErrProviderUnreachable is the provider not answering at all.
	ErrProviderUnreachable = errors.New("the provider could not be reached")
	// ErrProviderNonsense is the provider answering with something that is not
	// a completion — nearly always a proxy or an auth page in front of it.
	ErrProviderNonsense = errors.New("the provider answered with something else")
)

// Why names the reason for a log line, or "" when the error is not one of
// these. It is deliberately not exported as a type: this is a string for a
// human to read in a log, not a thing to switch on.
func Why(err error) string {
	switch {
	case errors.Is(err, ErrProviderRefused):
		return "refused"
	case errors.Is(err, ErrProviderUnreachable):
		return "unreachable"
	case errors.Is(err, ErrProviderNonsense):
		return "nonsense"
	default:
		return ""
	}
}

// Now is the only context sent on every call. Capacity is derived: the model is
// told "ok" or "low", never "wiped" and never a history — a signal rather than a
// diagnosis.
type Now struct {
	// Clock is wall time, "15:04". The model has no clock of its own.
	Clock string
	// PartOfDay is "morning", "afternoon" or "evening".
	PartOfDay string
	// Capacity is "ok" or "low".
	Capacity string
	// FreeUntil is minutes until the next fixed point, or nil when nothing is
	// coming. Nil is not "plenty of time" — it is "nothing was typed".
	FreeUntil *int
	// LandedBadly is the last few answers this person said did not land, in the
	// model's own words. Examples, never a count.
	LandedBadly []string
	// Knowing is what a weekly read of the record concluded about how this person
	// works. Shown to the model and never to the person mid-sentence — see knowsYou.
	//
	// Sentences rather than fields: a schema decides in advance what is worth knowing
	// about somebody.
	Knowing []string
}

// Turn is one thing said to the coach.
type Turn struct {
	// PersonID is whose budget this spends. Carried on the turn rather than
	// checked by the caller, so that no call site can forget the ceiling.
	PersonID int64
	// Kind is which surface asked: "chat", "sheet", "overwhelm". It reaches
	// the log and nothing else, and it is what makes "which surface produces
	// the answers that land badly" a question with an answer.
	Kind string
	// Room is which room this was said in: "everything", "notes", "chores", "at",
	// "tasks", "held", "kept". Empty means Buddy's own room, which is where
	// every turn was said before 28 August 2026.
	//
	// Stated by the surface rather than inferred here, for the same reason
	// CanOpen is: this package cannot know, and guessing would be a second
	// answer to a question the caller already has.
	//
	// It decides two things — which tools Buddy is given, and which
	// conversation he is carrying — so a wrong value is not a wrong label, it
	// is the wrong Buddy.
	Room string
	// Deep asks for the escalation tier. The caller decides, because the
	// caller is the only thing that knows whether this is a routine turn or
	// one where judgement matters. Overwhelmed() is what sets it today.
	Deep bool
	Now  Now
	// Said is what the person typed, verbatim. Never trimmed of meaning.
	Said string
	// Subject is what is on screen, when there is something: the offer's text,
	// the task being started. Empty when the turn is not about anything in
	// particular.
	Subject string
	// Recent is the last few exchanges, oldest first, and it is bounded by the
	// caller rather than by this package. See Window.
	Recent []Exchange
	// CanOpen says the surface that asked can draw one of the places. Chat cannot: a
	// place there would be the list the guard exists to refuse.
	//
	// The surface states it rather than this package inferring it from Kind, because
	// what a surface can draw is a fact about the surface.
	CanOpen bool
}

// Exchange is one round of a conversation, for the rolling window.
type Exchange struct {
	Said    string
	Replied string
	At      time.Time
}

// Window is how much conversation is carried between calls, and how long it
// survives. Three exchanges and half an hour: a coach that remembers the whole
// day grows a prompt through the day, and one that remembers nothing cannot hear
// "no, something else".
const (
	WindowSize = 3
	WindowAge  = 30 * time.Minute
)

// Reply is what came back, after the guard.
type Reply struct {
	// Text is what to show. It has already passed Guard, so a caller may
	// render it without checking anything.
	Text string
	// Model is which model produced it, for the log.
	Model string
	// InTokens and OutTokens are what it cost, for the budget.
	InTokens, OutTokens int
	// Did is what actually changed, in the application's words rather than
	// the model's. A model saying "done" is not evidence anything happened;
	// this is written after the write succeeded.
	Did []string
	// Propose is a thing the coach wants to do and may not do on its own, or
	// nil. The caller renders it as one press and applies it only if pressed.
	Propose *Proposal
	// Open is one of the places, when the coach asked for it to be shown. The caller
	// draws it. It needs no permission because it changes nothing — the worst a wrong
	// one costs is a scroll.
	Open string
}

// Coach is the surface the product asks. Methods are added as the phases that
// need them land; each one answers ErrUnavailable from NoCoach.
type Coach interface {
	// Answer is a conversational turn: the screen, `!coach`, and the overwhelm
	// turn.
	Answer(ctx context.Context, t Turn) (Reply, error)
	// Smaller breaks one thing into steps. It is the one method that returns
	// a list, and it is safe because nothing renders one: the sequence is
	// stored and handed back a step at a time.
	Smaller(ctx context.Context, personID int64, task, blocker string) ([]string, error)
	// Split proposes the separate things in one note. It returns strings and
	// nothing else — it has no way to write anything, which is decision 8
	// stated as a property rather than an intention.
	Split(ctx context.Context, personID int64, text string) ([]string, error)
	// ShouldInterrupt is a veto on something the rules already allowed, and
	// optional wording for it. It cannot cause an interruption that would not
	// otherwise have happened, because nothing else is ever passed to it — and
	// it fails open, so an absent coach leaves the nudge exactly as it was.
	ShouldInterrupt(ctx context.Context, personID int64, about string, n Now) (string, bool)
	// Reads answers what was typed into the box, says whether it was a thought
	// worth keeping or a question it has just answered, and names a place to
	// show when the words asked to see one.
	//
	// ErrUnavailable means the box behaves exactly as it did before this
	// existed: kept, and "Kept." said back. Every failure lands on the old
	// guarantee rather than on a lost thought.
	Reads(ctx context.Context, personID int64, said string, n Now) (string, bool, string, error)
	// Notice reads the board and writes at most two lines about what is on it,
	// each attached to one thing. ErrUnavailable means nothing is written,
	// which is also what a board with nothing worth saying about it produces —
	// the two are the same outcome on purpose.
	Notice(ctx context.Context, personID int64, things []Thing, refused []string) ([]Note, error)
	// Learn reads the record of the conversation back and says what it shows
	// about how this person works. Once a week, and everything about it is
	// optional: ErrUnavailable leaves Squirrel knowing whatever it knew, which
	// for a month was nothing at all.
	Learn(ctx context.Context, personID int64, record []string) ([]string, error)
}

// NoCoach is the zero value and the default build. It is not a stub for tests
// — it is the shipping configuration whenever a key is absent, and the reason
// every caller has a deterministic path.
type NoCoach struct{}

func (NoCoach) Answer(context.Context, Turn) (Reply, error) {
	return Reply{}, ErrUnavailable
}

func (NoCoach) Notice(context.Context, int64, []Thing, []string) ([]Note, error) {
	return nil, ErrUnavailable
}

func (NoCoach) Learn(context.Context, int64, []string) ([]string, error) {
	return nil, ErrUnavailable
}

func (NoCoach) Reads(context.Context, int64, string, Now) (string, bool, string, error) {
	return "", true, "", ErrUnavailable
}

func (NoCoach) Smaller(context.Context, int64, string, string) ([]string, error) {
	return nil, ErrUnavailable
}

func (NoCoach) Split(context.Context, int64, string) ([]string, error) {
	return nil, ErrUnavailable
}

// NoCoach lets every interruption through, unchanged. Alone among these it
// does not answer "unavailable": there is nothing to be unavailable for, since
// the rules already decided and the nudge worked on its own for months before
// any of this.
func (NoCoach) ShouldInterrupt(context.Context, int64, string, Now) (string, bool) {
	return "", true
}

// Trim keeps the newest exchanges that are still recent enough to be about
// now. It lives here rather than in a caller so both surfaces cannot disagree
// about how long a conversation lasts.
func Trim(recent []Exchange, now time.Time) []Exchange {
	fresh := make([]Exchange, 0, WindowSize)
	for _, e := range recent {
		if now.Sub(e.At) < WindowAge {
			fresh = append(fresh, e)
		}
	}
	if len(fresh) > WindowSize {
		fresh = fresh[len(fresh)-WindowSize:]
	}
	return fresh
}

// The rooms this package knows the names of, and it is a copy: internal/coach
// must not import internal/web, and Buddy has to be able to say where he is.
//
// Everything is deliberately absent. It is not a room he is confined to — it is
// where he is, and every other room is the narrowing.
//
// TestTheRoomNamesAgreeWithTheCoach fails when the two lists drift, because
// two lists of the same four names is one list that goes stale.
var roomNames = map[string]string{
	"notes":  "the notes",
	"chores": "the chores",
	"at":     "the agenda",
	"tasks":  "the tasks",
}

// RoomName is what a room is called, or empty for Buddy's own and for anything
// this package has never heard of.
func RoomName(key string) string { return roomNames[key] }

// RoomKeys is every room this package narrows in, in no particular order.
// Buddy's own is not among them.
func RoomKeys() []string {
	out := make([]string, 0, len(roomNames))
	for key := range roomNames {
		out = append(out, key)
	}
	return out
}
