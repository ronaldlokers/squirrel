package coach_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/coach"
)

// The confirmation policy, pinned. The test behind every line here is one
// question: is this already a button the person could have pressed, and does
// one press undo it?

type fakeHands struct {
	did []string
	err error
}

// note records which tool ran and hands back what the real one would: the name
// of the thing that was acted on, so the surface can say what changed in the
// application's words.
func (h *fakeHands) note(tool, what string) (string, error) {
	if h.err != nil {
		return "", h.err
	}
	h.did = append(h.did, tool)
	return what, nil
}

func (h *fakeHands) Complete(_ context.Context, _, id int64) (string, error) {
	return h.note("complete", "ring the vet")
}

func (h *fakeHands) CompleteChore(_ context.Context, _, id int64) (string, error) {
	return h.note("complete_chore", "put the bins out")
}

func (h *fakeHands) StartTimer(_ context.Context, _ int64, _ string, _ int) error {
	_, err := h.note("start_timer", "")
	return err
}

func (h *fakeHands) Refuse(_ context.Context, _ int64, _ string, _ int64) error {
	_, err := h.note("refuse", "")
	return err
}

func (h *fakeHands) SnoozeChore(_ context.Context, _, _ int64, _ int) (string, error) {
	return h.note("snooze_chore", "put the bins out")
}

func (h *fakeHands) CreateTask(_ context.Context, _ int64, _ string) error {
	_, err := h.note("create_task", "")
	return err
}

func actingFor(api *toolAPI, f *fakeFacts, h *fakeHands, log *fakeLog) *coach.Provider {
	p := coach.NewProvider(api.server.URL, "sk-test", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: log, CeilingFor: coach.FlatCeiling(10_000_000)})
	p.Clock = func() time.Time { return august }
	p.Facts = f
	p.Hands = h
	return p
}

func aTurn() coach.Turn {
	return coach.Turn{PersonID: 1, Kind: "sheet", Said: "I did the bins"}
}

// Most of what someone says to a coach wants an answer rather than an action.
// A model given tools will use them, so saying nothing happened has to be an
// easy, explicit thing to do.
func TestATurnThatOnlyTalksChangesNothing(t *testing.T) {
	h := &fakeHands{}
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{
		"text": "Good. That is the one that was bothering you.",
	})))

	reply, err := actingFor(api, &fakeFacts{}, h, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Equal(t, "Good. That is the one that was bothering you.", reply.Text)
	require.Empty(t, h.did)
	require.Nil(t, reply.Propose)
}

// The six that pass the test run when asked, and the surface says what changed
// in the application's words rather than repeating the model's claim about it.
func TestADirectWriteRunsAndSaysWhatChanged(t *testing.T) {
	f := &fakeFacts{work: []coach.Work{{ID: 3, Kind: "chore", Text: "put the bins out"}}}
	h := &fakeHands{}
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(
			call("b", "complete_chore", map[string]any{"chore_id": 3}),
			call("c", "say", map[string]any{"text": "Done."}),
		),
	)

	reply, err := actingFor(api, f, h, &fakeLog{}).Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Equal(t, []string{"complete_chore"}, h.did)
	require.Equal(t, []string{"put the bins out is done"}, reply.Did)
	require.Equal(t, "Done.", reply.Text)
}

// The same check the picker's own decision rests on: a model can act badly, it
// cannot act on something it invented.
func TestAWriteOnSomethingItWasNeverShownIsRefused(t *testing.T) {
	h := &fakeHands{}
	api := newToolAPI(t,
		turnOf(call("a", "complete", map[string]any{"item_id": 99})),
		turnOf(call("b", "say", map[string]any{"text": "I could not."})),
	)

	reply, err := actingFor(api, &fakeFacts{}, h, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Empty(t, h.did, "it acted on something it was never shown")
	require.Empty(t, reply.Did)
}

// A model that thinks it acted will say so, and saying so when nothing
// happened is the worst failure available here. So a failed write is reported
// back rather than hidden.
func TestAFailedWriteIsToldToTheModelAndNeverClaimed(t *testing.T) {
	f := &fakeFacts{work: []coach.Work{{ID: 3, Kind: "chore", Text: "put the bins out"}}}
	h := &fakeHands{err: errors.New("no database")}
	// Both in one turn, so the tool results reach the next request and the
	// test can read what the model was actually told.
	api := newToolAPI(t,
		turnOf(
			call("a", "open_work", nil),
			call("b", "complete_chore", map[string]any{"chore_id": 3}),
		),
		turnOf(call("c", "say", map[string]any{"text": "That did not work."})),
	)

	reply, err := actingFor(api, f, h, &fakeLog{}).Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Empty(t, reply.Did, "a failed write was reported as having happened")

	sent, _ := api.sent[1]["messages"].([]any)
	var toldIt bool
	for _, m := range sent {
		msg, _ := m.(map[string]any)
		if msg["role"] != "tool" {
			continue
		}
		if content, _ := msg["content"].(string); content == `{"done":false,"why":"it did not work"}` {
			toldIt = true
		}
	}
	require.True(t, toldIt, "the model was not told the write failed")
}

func TestTheFourThatNeedAskingAreOnlyProposed(t *testing.T) {
	for _, do := range []string{"moment", "chore", "retire", "drop"} {
		h := &fakeHands{}
		api := newToolAPI(t,
			turnOf(call("a", "propose", map[string]any{
				"do": do, "said": "Shall I keep this?", "text": "dentist", "at": "14:30",
				"every": "every 2 weeks", "ref_id": 3,
			})),
			turnOf(call("b", "say", map[string]any{"text": "Say the word."})),
		)

		reply, err := actingFor(api, &fakeFacts{}, h, &fakeLog{}).
			Answer(context.Background(), aTurn())
		require.NoError(t, err)
		require.Empty(t, h.did, "%q was done rather than asked about", do)
		require.NotNil(t, reply.Propose, "%q produced no proposal", do)
		require.Equal(t, do, reply.Propose.Do)
	}
}

// A kind that is not one of the four is not a proposal. The vocabulary is a
// map so an unknown value is a lookup miss rather than a default branch.
func TestAnUnknownProposalIsNoProposal(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "propose", map[string]any{"do": "delete", "said": "Shall I?"})),
		turnOf(call("b", "say", map[string]any{"text": "Never mind."})),
	)

	reply, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Nil(t, reply.Propose)
	require.False(t, coach.KnownProposal("delete"))
	require.False(t, coach.KnownProposal("reword"))
}

// The sentence someone reads before pressing something is held to the same
// shape as everything else they read here.
func TestAProposalThatArrivesAsAPlanIsThrownAway(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "propose", map[string]any{
			"do": "chore", "said": "Here's what I'd do:\n- one\n- two", "text": "bins",
		})),
		turnOf(call("b", "say", map[string]any{"text": "Never mind."})),
	)

	reply, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Nil(t, reply.Propose)
}

// A timer is between one minute and three hours. Bounded in code, because a
// model asked not to start a nine-hour timer is a model that can.
func TestATimerOutsideItsBoundsIsRefused(t *testing.T) {
	h := &fakeHands{}
	api := newToolAPI(t,
		turnOf(call("a", "start_timer", map[string]any{"label": "the kitchen", "minutes": 600})),
		turnOf(call("b", "say", map[string]any{"text": "I could not."})),
	)

	_, err := actingFor(api, &fakeFacts{}, h, &fakeLog{}).Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Empty(t, h.did)
}

// A loop is a place for a model to talk itself into acting, so there are two
// rounds and no more.
func TestATurnThatNeverAnswersIsNoAnswer(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "open_work", nil)),
		turnOf(call("b", "lately", nil)),
		turnOf(call("c", "say", map[string]any{"text": "too late"})),
	)

	_, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Len(t, api.sent, 2)
}

// The guard applies to what say() carries, exactly as it applies to prose.
func TestSayingAPlanIsRefused(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{
		"text": "Here's the plan:\n- open it\n- read it\n- reply",
	})))

	_, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.ErrorIs(t, err, coach.ErrUnavailable)
}

func TestAPlanShapedProseReplyDoesNotLogTheModelsWords(t *testing.T) {
	logs := captureLogs(t)
	api := newToolAPI(t, said("Here's the plan:\n- open the letter\n- ring the tax office\n- pay it"))

	_, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.ErrorIs(t, err, coach.ErrUnavailable)

	require.NotContains(t, logs.String(), "tax office")
	require.Contains(t, logs.String(), "said_len")
}

// Without hands the turn can only talk, which is what every turn did before
// the phase that changed it.
func TestWithoutHandsATurnOffersNoTools(t *testing.T) {
	api := newFakeAPI(t, "Start with the envelope.")
	p := coach.NewProvider(api.server.URL, "sk", "gpt-5.6-luna", "gpt-5.6-terra",
		coach.Budget{Log: &fakeLog{}})
	p.Clock = func() time.Time { return august }

	reply, err := p.Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Equal(t, "Start with the envelope.", reply.Text)
	require.NotContains(t, api.requests[0], "tools")
}

func TestActingMakesNoCallWhenOverBudget(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "hello"})))
	_, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{spent: 10_000_000}).
		Answer(context.Background(), aTurn())
	require.ErrorIs(t, err, coach.ErrUnavailable)
	require.Empty(t, api.sent)
}

// The three that are never available, asserted as absences: rewriting your own
// words is what !fix is for; how you feel is said by you; nothing deletes.
func TestTheRefusedToolsAreNotOffered(t *testing.T) {
	api := newToolAPI(t, turnOf(call("a", "say", map[string]any{"text": "hello"})))
	_, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)

	tools, ok := api.sent[0]["tools"].([]any)
	require.True(t, ok)
	named := map[string]bool{}
	for _, tool := range tools {
		fn, _ := tool.(map[string]any)["function"].(map[string]any)
		name, _ := fn["name"].(string)
		named[name] = true
	}

	for _, refused := range []string{"reword", "checkin", "delete", "drop", "create_moment"} {
		require.False(t, named[refused], "%q is offered to the model", refused)
	}
	// And the six that are.
	for _, allowed := range []string{
		"complete", "complete_chore", "start_timer", "refuse", "snooze_chore", "create_task",
	} {
		require.True(t, named[allowed], "%q is missing", allowed)
	}
}

// Asking to *see* something is not asking about it, and the coach could not do
// it: Guard refuses a list and the brief is two sentences, so "show me the
// tasks" got an honest "I cannot" while the menu did it in one press. It can
// press the same thing now.
func TestAskingToSeeAPlaceOpensIt(t *testing.T) {
	api := newToolAPI(t,
		turnOf(
			call("a", "open", map[string]any{"where": "tasks"}),
			call("b", "say", map[string]any{"text": "Here they are."}),
		),
	)
	turn := aTurn()
	turn.CanOpen = true
	turn.Said = "can you show me the tasks"

	reply, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), turn)
	require.NoError(t, err)
	require.Equal(t, "tasks", reply.Open)
	require.Equal(t, "Here they are.", reply.Text)
	require.True(t, offers(t, api.sent[0])["open"], "the tool was not offered")
}

// Only where there is something to draw it on. Chat has no cards: a place
// there would be the list the guard exists to refuse, so the tool is not even
// offered and a model that names it anyway is refused.
func TestASurfaceThatCannotDrawAPlaceIsNotOfferedOne(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "open", map[string]any{"where": "tasks"})),
		turnOf(call("b", "say", map[string]any{"text": "I cannot show you that here."})),
	)

	reply, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), aTurn())
	require.NoError(t, err)
	require.Empty(t, reply.Open)

	// And the tool was never on the table.
	require.False(t, offers(t, api.sent[0])["open"], "the tool was offered where it cannot be drawn")
}

// offers is which tools a request put on the table.
func offers(t *testing.T, sent map[string]any) map[string]bool {
	t.Helper()
	tools, ok := sent["tools"].([]any)
	require.True(t, ok)
	named := map[string]bool{}
	for _, tool := range tools {
		fn, _ := tool.(map[string]any)["function"].(map[string]any)
		name, _ := fn["name"].(string)
		named[name] = true
	}
	return named
}

// A name that is not one of the six is a lookup miss, not a default branch.
func TestAPlaceThatDoesNotExistIsRefused(t *testing.T) {
	api := newToolAPI(t,
		turnOf(call("a", "open", map[string]any{"where": "inbox"})),
		turnOf(call("b", "say", map[string]any{"text": "You have no inbox."})),
	)
	turn := aTurn()
	turn.CanOpen = true

	reply, err := actingFor(api, &fakeFacts{}, &fakeHands{}, &fakeLog{}).
		Answer(context.Background(), turn)
	require.NoError(t, err)
	require.Empty(t, reply.Open)
}
