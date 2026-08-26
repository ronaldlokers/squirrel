package boot

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The same check `Push` earned the hard way: `SchedulerOptions` has a `Learn`
// field, `KnowingTick` returns immediately when it is nil, and a field left
// unset in a struct literal is invisible — the symptom is Squirrel simply not
// knowing anything about you, which is indistinguishable from a quiet fortnight.
//
// Deliberately about the option and not about what a pass concludes. What
// happens after the scheduler calls it is the coach's business.
func TestTheSchedulerIsGivenAWayToLearn(t *testing.T) {
	opts := schedulerOptionsFor(schedulerWiring{
		personID:       1,
		conversationID: "9",
		learn: func(context.Context, int64, []string) ([]string, error) {
			return nil, nil
		},
	})

	require.NotNil(t, opts.Learn,
		"the scheduler cannot learn without one, and its own nil check makes that silent")
}

// And nil stays meaningful. With no key there is nothing to read the record
// back with, and a Buddy who knows nothing about you is the Buddy that shipped
// for a month — a supported state rather than a failure, so it must not be
// papered over with a function that cannot work.
func TestNoCoachMeansNoLearnerRatherThanABrokenOne(t *testing.T) {
	require.Nil(t, learner(coach.NoCoach{}),
		"a build with no key was given something to call")

	opts := schedulerOptionsFor(schedulerWiring{personID: 1, conversationID: "9"})
	require.Nil(t, opts.Learn)
}

// With a coach there is one, and it is the coach's own method rather than a
// wrapper that could quietly answer something else.
func TestWithACoachThereIsALearner(t *testing.T) {
	require.NotNil(t, learner(stubCoach{}))
}

// stubCoach is a coach that is not NoCoach. Every method answers
// ErrUnavailable, which is what the real one does when it cannot reach a model
// — what is being tested here is the type check, not a behaviour.
type stubCoach struct{ coach.NoCoach }

var _ coach.Coach = stubCoach{}

// And what the prompt does with it. The observations reach the model or they
// change nothing at all, and nowFor is the one place that decides.
func TestWhatIsKnownReachesTheModel(t *testing.T) {
	require.Contains(t, coach.System(coach.Now{
		Knowing: []string{"Phone calls get done; forms get put off."},
	}, "chat"), "Phone calls get done")
}

var _ = squirrel.Learner(nil)

// The third field to earn this check. `captureHandler` returns the old
// behaviour when `Reads` is nil, so a field left unset in the options literal
// is a box that silently goes back to being a filing cabinet — and the symptom
// is Buddy saying "Kept.", which is exactly what he said for a month.
func TestTheScreenIsGivenAWayToReadTheBox(t *testing.T) {
	require.Nil(t, reader(coach.NoCoach{}, nil),
		"a build with no key was given something to call")
	require.NotNil(t, reader(stubCoach{}, nil))
}

// The fourth field to earn this check. `whatBuddyMakesOfIt` falls back to the
// rule when it is nil, so a field left unset is every capture going abroad
// again — which is the defect this whole seam exists to fix, and it would be
// invisible except on the bill.
func TestTheScreenIsGivenTheHouse(t *testing.T) {
	require.Nil(t, housed(nil), "a build with no house was given something to call")
	require.NotNil(t, housed(coach.NewHouse("http://the-house", "a small one")))
}

func TestNoAddressMeansNoHouse(t *testing.T) {
	require.Nil(t, coach.NewHouse("", "a small one"))
}

// The rule is compiled in and needs nothing. The other two are fields, and a
// field in an inline struct literal is exactly what `Push` cost three releases
// to learn about: nothing warns, nothing fails, and the symptom is a product
// that quietly does less.
//
// This one earned it immediately. `AskedAQuestion` was written, lost to a
// stray edit, and found only because a mutation went looking for it — the
// suite was green with every capture going abroad again, which is the defect
// the whole seam exists to fix.
func TestTheBoxIsGivenItsThreeTiers(t *testing.T) {
	with := readingWiring(stubCoach{}, nil, coach.NewHouse("http://the-house", "a small one"))

	require.NotNil(t, with.Reads, "there is nobody to answer a question")
	require.NotNil(t, with.AskedAQuestion, "every capture goes abroad to be judged")

	// And the nils stay meaningful: no key and no house is the configuration
	// this shipped with, and the rule answers alone.
	without := readingWiring(coach.NoCoach{}, nil, nil)
	require.Nil(t, without.Reads)
	require.Nil(t, without.AskedAQuestion)
}

// sayingCoach answers with a place to open, so the seam between the coach's
// reply and the screen's answer can be read rather than assumed.
type sayingCoach struct {
	coach.NoCoach
	reply coach.Reply
	saw   coach.Turn
}

func (c *sayingCoach) Answer(_ context.Context, t coach.Turn) (coach.Reply, error) {
	c.saw = t
	return c.reply, nil
}

// The fourth field to earn this check, and it is the same shape as the other
// three: `Open` is a field in an inline literal in `coachWeb`, nothing warns if
// it is dropped, and the symptom is Buddy saying "here they are" above nothing
// at all — a reply that claims to have done something it did not do, which is
// the worst failure available here.
func TestTheScreenIsGivenThePlaceToOpen(t *testing.T) {
	c := &sayingCoach{reply: coach.Reply{Text: "Here they are.", Open: "tasks"}}
	ask, _, _, _ := coachWeb(c, nil, coach.NewConversations())

	answer, err := ask(context.Background(), 1, "thread", "show me the tasks", "")
	require.NoError(t, err)
	require.Equal(t, "tasks", answer.Open, "the screen was told nothing to open")
}

// And the screen is the surface allowed to ask for one. Chat is not: a place
// there would be the list the guard exists to refuse.
func TestOnlyTheScreenMayBeOfferedAPlace(t *testing.T) {
	c := &sayingCoach{reply: coach.Reply{Text: "Here they are."}}
	ask, _, _, _ := coachWeb(c, nil, coach.NewConversations())
	_, err := ask(context.Background(), 1, "thread", "show me the tasks", "")
	require.NoError(t, err)
	require.True(t, c.saw.CanOpen, "the screen cannot draw a place")

	chat := &sayingCoach{reply: coach.Reply{Text: "Here they are."}}
	_, err = asker(chat, nil, coach.NewConversations(), false)(
		context.Background(), 1, "chat", "show me the tasks", "")
	require.NoError(t, err)
	require.False(t, chat.saw.CanOpen, "chat was offered a place it cannot draw")
}
