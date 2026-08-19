package squirrel

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

// Sender posts a message the system initiated. A func type rather than an
// interface, so internal/boot can pass a transport's Send straight in and this
// package still imports nothing from internal/transport.
type Sender func(ctx context.Context, conversationID, text string) error

// undoWindow is how long `nvm` can reach back to remove a chore the matcher
// created from what was meant as a note.
const undoWindow = 10 * time.Minute

type Applier struct {
	store *Store
	// send is the phase 2 plain-text surface. apply() no longer calls it —
	// every reply is a Message now, sent through chat — but the field and
	// NewApplier's parameter stay so boot.go (rewired in a later task) and
	// phase 2 callers still compile.
	send    Sender
	chat    Chat
	onError func(error)
	// pending is the id of the prompt recorded earlier in this same apply()
	// call, if any — RecordPrompt commits before the message carrying its
	// buttons is sent, the same ordering the scheduler uses and for the same
	// reason, so the id has to survive the few lines between the two. Reset
	// to zero at the top of every apply().
	pending int64
	// nudger optionally attaches a nudge to a capture — set after
	// construction via SetNudger, since the Applier and the Scheduler each
	// need the other and boot builds them in that order.
	nudger func(ctx context.Context, now time.Time, why NudgeReason) error
}

// SetNudger supplies the callback that may attach a nudge to a capture. It is
// set after construction because the Applier and the Scheduler each need the
// other, and boot builds them in that order.
func (a *Applier) SetNudger(n func(context.Context, time.Time, NudgeReason) error) {
	a.nudger = n
}

// NewApplier's send parameter is a vestige of phase 2, kept only so callers
// built before Chat existed still compile — see the comment on Applier.send.
// chat is what every reply and receipt actually goes through now.
func NewApplier(store *Store, send Sender, chat Chat, onError func(error)) *Applier {
	if onError == nil {
		onError = func(error) {}
	}
	return &Applier{store: store, send: send, chat: chat, onError: onError}
}

// Apply runs after a capture has landed in Postgres. The raw text is already
// stored verbatim, so everything here is a derived view over it and nothing
// here can lose a thought.
//
// A panic anywhere below is recovered and reported as an error rather than
// left to propagate. It escaped once already — a byte-length bug in Match's
// chore-name parsing — and rode out through Drain.one and Drain.Run to exit
// the whole process. By then the spool file was already gone and the row
// already committed, so there was nothing left to retry, and every later
// digest attempt re-ran Match over that same row via CapturesSince and
// panicked the scheduler too, forever. A derived view failing must never be
// able to take capture down with it again.
func (a *Applier) Apply(ctx context.Context, item Item, personID *int64) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("applier panicked: %v", r)
		}
	}()
	return a.apply(ctx, item, personID)
}

func (a *Applier) apply(ctx context.Context, item Item, personID *int64) error {
	a.pending = 0

	// Chores belong to a person. An unresolved identity means we do not know
	// whose they would be, so nothing is applied — the capture is already safe.
	if personID == nil || item.ConversationID == nil {
		return nil
	}

	// kind gates nudgeBack below: only a genuine capture ever carries a nudge
	// back (see nudgeBack's own comment). A tap never sets it — see the
	// action branch — so it stays its zero value, which is never
	// IntentCapture.
	var kind IntentKind

	// An action is not a thought and never reaches the matcher. It is checked
	// first so that a tap can never be reinterpreted as text. ParseAction
	// matches on text alone, so it cannot by itself tell a genuine tap from
	// someone typing the same shape into the room — isActionPayload is what
	// makes that distinction, from the payload the transport actually sent.
	//
	// Both branches below funnel into the same tail: tick runs once handling
	// has succeeded, whether that handling was a tap resolved against a
	// prompt or a plain capture that reached Postgres. tick itself is the one
	// that knows a tap earns no ✅ — this join point does not need to.
	if in, ok := ParseAction(item.RawText); ok && isActionPayload(item.Payload) {
		if err := a.applyAction(ctx, in, *personID); err != nil {
			return err
		}
	} else {
		intent := matchFn(item.RawText)
		kind = intent.Kind
		m, err := a.replyFor(ctx, intent, *personID, *item.ConversationID)
		if err != nil {
			return err
		}
		if m.Text != "" {
			messageID, err := a.sendMessage(ctx, *item.ConversationID, m)
			if err != nil {
				return err
			}
			if a.pending != 0 {
				if messageID == "" {
					// The transport reported success but returned no id to
					// hang the buttons off — see chatVia's messageIDFrom.
					// Still mark the prompt delivered below; just say, once,
					// that it can never be disabled and no tap can resolve
					// back to it, rather than storing a lie.
					a.onError(fmt.Errorf("prompt %d delivered with no addressable message id", a.pending))
				}
				if err := a.store.MarkPromptSent(ctx, a.pending, messageID, time.Now()); err != nil {
					return err
				}
				// Only a surface that carries buttons closes the one before
				// it, and only a numbered one at that. A define names one
				// chore and takes no position, so closing the digest on its
				// account would retire buttons the morning's list still owns.
				//
				// The len(m.Actions) test is why `!notes` no longer kills the
				// day's nudge. closePrevious exists to keep exactly one live
				// *button* surface; a pile listing deliberately carries none
				// (see NotesMessage), and a tap resolves through its own
				// message id, so leaving the nudge live creates no ambiguity.
				// Closing it bought nothing and cost the day's chore: the ✅
				// went grey the moment you looked at your own notes.
				if intent.Kind != IntentDefine && len(m.Actions) > 0 {
					closePrevious(ctx, a.store, a.chat, a.onError, *personID, a.pending)
				}
			}
		}
	}

	a.tick(ctx, item)
	a.nudgeBack(ctx, kind)
	return nil
}

// sendMessage sends through Chat when the transport supports it, and falls
// back to the phase 2 plain-text Sender otherwise — Boost is already guarded
// this way in tick, and Update is already guarded this way in closePrevious;
// Send was the one field this method still called unconditionally, which is
// exactly why every phase 2 test that reaches this path had to build a Chat
// around its Sender in the first place. Falling back here instead makes
// "degrade to phase 2 behaviour when Chat carries no Send" true by
// construction, not by test scaffolding or deployment discipline, and gives
// the send field a reason to still exist.
func (a *Applier) sendMessage(ctx context.Context, conversationID string, m Message) (string, error) {
	if a.chat.Send == nil {
		return "", a.send(ctx, conversationID, m.Text)
	}
	return a.chat.Send(ctx, conversationID, m)
}

// tick is the second half of the receipt. 👀 said the thought was on disk; ✅
// says it has been handled — the drain reached Postgres and this ran. Both
// stay: the pair is a visible trail of two stages that genuinely happened,
// rather than one state overwriting its own history.
//
// Fail-open, like every part of the receipt. A boost that cannot be created
// never changes whether the capture was stored.
func (a *Applier) tick(ctx context.Context, item Item) {
	if a.chat.Boost == nil || item.ExternalID == nil || item.ConversationID == nil {
		return
	}
	if isTap(item) {
		// A tap is not a message in the room; there is nothing to react to.
		return
	}
	if err := a.chat.Boost(ctx, *item.ConversationID, *item.ExternalID, "✅"); err != nil {
		a.onError(fmt.Errorf("ticking %s: %w", *item.ExternalID, err))
	}
}

// Reactions are what a completion earns. Small, immediate, non-cumulative and
// unpredictable — intermittent reinforcement is the strongest schedule there
// is, and it is the same mechanism that defeats habituation, so one change
// serves both.
//
// What makes a streak punish is not the reward but the counter that resets:
// loss aversion makes losing hurt about twice as much as the equivalent gain
// pleases. Nothing here accrues, so there is nothing to lose.
//
// These are NOT the 👀/✅ receipt, which reports whether the thought is on disk
// and whether it reached Postgres. That pair is information and must never
// vary — randomising it would turn the one honest signal about durability into
// decoration.
var Reactions = []string{"🎉", "✨", "🙌", "💫", "🌟"}

// react acknowledges a completion on the message that asked for it. Fail-open:
// a reaction that cannot be sent is cosmetic, and must never turn a recorded
// completion into an error.
func (a *Applier) react(ctx context.Context, prompt Prompt) {
	if a.chat.Boost == nil || prompt.ExternalMessageID == "" {
		return
	}
	pick := Reactions[rand.Intn(len(Reactions))]
	if err := a.chat.Boost(ctx, prompt.ConversationID, prompt.ExternalMessageID, pick); err != nil {
		a.onError(fmt.Errorf("reacting to a completion: %w", err))
	}
}

// nudgeBack rides a nudge home on a message the person sent. Fail-open, like
// every other outbound: a nudge that cannot be sent must never turn a stored
// capture into an error, and the budget means a missed one is simply not
// replaced today.
//
// Gated to a plain capture — kind == IntentCapture — not merely "not a tap".
// apply() already excludes IntentDefine from closePrevious for the same
// reason, and the spec's own trigger table says "any inbound capture", not
// "any inbound message". A command reply (?, done, stop 1, ...) opens its own
// numbered surface — a list, a confirmation — and nudgeBack riding in right
// behind it would open a second one a beat later, disabling the buttons the
// reply itself just printed: typing `?` would list three chores and then,
// via Nudge's own closePrevious, immediately un-list them, so `done 3`
// answers "I don't have a line 3" until the next `?` spends the day's budget
// and it happens not to recur. A tap never reaches here as a capture either —
// see apply()'s action branch, which leaves kind at its zero value.
func (a *Applier) nudgeBack(ctx context.Context, kind IntentKind) {
	if a.nudger == nil || kind != IntentCapture {
		return
	}
	if err := a.nudger(ctx, time.Now(), NudgeFromMessage); err != nil {
		a.onError(fmt.Errorf("nudging after a capture: %w", err))
	}
}

func (a *Applier) replyFor(ctx context.Context, in Intent, personID int64, conversationID string) (Message, error) {
	switch in.Kind {
	case IntentDefine:
		c, err := a.store.UpsertChoreAsking(ctx, personID, in.Name, in.Every, DefaultTolerance(in.Every), in.Ask)
		if err != nil {
			return Message{}, err
		}
		// A define is a standalone surface: it names one chore, it is never
		// numbered, and it does not close the digest's buttons.
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "define", time.Now(), nil, []Chore{c})
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		return DefinedMessage(c), nil

	case IntentComplete:
		return a.complete(ctx, in, personID, conversationID)

	case IntentStop:
		// Resolved through LineAtPosition rather than ChoreAtPosition so a
		// note line gets the true answer. ChoreAtPosition simply fails to
		// join one, which made `stop 1` on the pile say "I don't have a line
		// 1" about a line printed a second earlier — and noSuchLine's whole
		// job is to mean "that surface is gone".
		line, ok, err := a.store.LineAtPosition(ctx, personID, in.Position)
		if err != nil {
			return Message{}, err
		}
		if !ok {
			return noSuchLine(in.Position), nil
		}
		if line.Item == nil && line.Chore == nil {
			return noSuchLine(in.Position), nil
		}
		if line.Chore == nil {
			return Message{Text: fmt.Sprintf("Line %d is a note, not a chore. Try drop %d.", in.Position, in.Position)}, nil
		}
		c := *line.Chore
		if err := a.store.DeactivateChore(ctx, c.ID); err != nil {
			return Message{}, err
		}
		return Message{Text: "Stopped " + c.Name + "."}, nil

	case IntentQuery:
		chores, err := a.store.ActiveChores(ctx, personID)
		if err != nil {
			return Message{}, err
		}
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, chores)
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		return ListMessage(chores), nil

	case IntentKeep:
		return a.triage(ctx, in.Position, personID, ItemKept, "Kept —")

	case IntentDrop:
		// A bare `nvm` undoes a chore the matcher just made from a note, which
		// is what phase 2 built and what it still means. `drop 2` is the
		// numbered form and drops that note. They share a Kind and are told
		// apart by Position, which is safe because none of the bare forms —
		// "nvm", "forget it", "never mind" — carries a number.
		if in.Position > 0 {
			return a.triage(ctx, in.Position, personID, ItemDropped, "Dropped —")
		}
		return a.undo(ctx, personID)

	case IntentCommand:
		return a.command(ctx, in, personID, conversationID)
	}

	// IntentCapture: the squirrel already went out in the HTTP response.
	return Message{}, nil
}

// listCap is how many lines a `!notes` or `!find` reply prints.
//
// Ten, not twelve: MaxActions is twelve because that is the fork's limit on
// buttons, and these lists carry none. Ten is a screenful on a phone, which is
// the constraint that actually applies here.
const listCap = 10

// intervalSentinel stands in for a chore name while an interval is parsed on
// its own. Any word that is not a unit works; this one says why it is there if
// it ever surfaces in an error.
const intervalSentinel = "chore-name-placeholder"

// command answers a ! command.
//
// Every branch that prints a numbered list records a prompt and sets pending,
// which is what makes `done 2` resolve against it: apply() marks the prompt
// delivered once the send succeeds and closes the previous numbered surface.
// Getting that for free is the reason these lists are prompts at all.
func (a *Applier) command(ctx context.Context, in Intent, personID int64, conversationID string) (Message, error) {
	switch in.Command {
	case "notes":
		if in.Arg != "" {
			// "!notes to self: the boiler man comes tuesday" is the one
			// command shape that reads like a sentence — the spec names
			// "notes to self" as its motivating example for why commands need
			// a prefix at all. Printing the pile and dropping the sentence is
			// the worst of both. Falling through to the unknown-command reply
			// at least says something happened.
			break
		}
		items, more, err := a.store.OpenItems(ctx, personID, listCap)
		if err != nil {
			return Message{}, err
		}
		return a.numbered(ctx, "notes", items, more, personID, conversationID)

	case "find":
		if in.Arg == "" {
			// An empty search reads as a mistake, not as a request for all of
			// it — and answering it with the whole pile would be the counting
			// behaviour by another route.
			return Message{Text: "What should I look for? Try !find boiler."}, nil
		}
		items, more, err := a.store.SearchItems(ctx, personID, in.Arg, listCap)
		if err != nil {
			return Message{}, err
		}
		if len(items) == 0 {
			return Message{Text: fmt.Sprintf("Nothing matching %q.", in.Arg)}, nil
		}
		return a.numbered(ctx, "find", items, more, personID, conversationID)

	case "chores":
		return a.replyFor(ctx, Intent{Kind: IntentQuery}, personID, conversationID)

	case "chore":
		return a.promote(ctx, in.Arg, personID)

	case "retire":
		return a.retire(ctx, in.Arg, personID)

	case "undo":
		return a.undoLast(ctx, personID)

	case "snooze":
		return a.snooze(ctx, in.Arg, personID)

	case "did":
		return a.did(ctx, in.Arg, personID)

	case "mood", "feel":
		return a.checkin(ctx, in.Arg, personID)

	case "fix":
		return a.fix(ctx, in.Arg, personID)

	case "help":
		return HelpMessage(), nil
	}

	// An unknown command is a typo, and a typo answered with 👀 would be filed
	// as a note along with the correction. Say what exists instead.
	m := HelpMessage()
	m.Text = fmt.Sprintf("I don't know !%s.\n\n%s", in.Command, m.Text)
	return m, nil
}

// promote turns note n into a recurring chore: `!chore 1 every 2 weeks`.
//
// The note's own text becomes the chore's name, and the note becomes `done` —
// it did its job by turning into something that comes back on its own.
//
// No column links the two. There is exactly one question that would read it
// ("where did this chore come from") and no second, and the rule here is two
// concrete cases before an interface. If a reason appears, it is a migration.
func (a *Applier) promote(ctx context.Context, arg string, personID int64) (Message, error) {
	position, rest, _ := strings.Cut(arg, " ")
	n, err := strconv.Atoi(position)
	if err != nil || n < 1 {
		return Message{Text: "Which line? Try !chore 1 every 2 weeks."}, nil
	}

	line, ok, err := a.store.LineAtPosition(ctx, personID, n)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(n), nil
	}
	if line.Item == nil {
		return Message{Text: fmt.Sprintf("Line %d is already a chore.", n)}, nil
	}

	// ParseEvery wants "every <interval> <name>" and returns the name out of
	// the same string, because on its usual path the name is what follows the
	// interval. Here the name is the note, so a sentinel is appended and only
	// the duration is kept. Reusing ParseEvery rather than reaching into its
	// regex keeps one definition of what an interval means.
	//
	// The sentinel is a fixed word, NOT the note's own text, and that is a
	// fix rather than a detail. Appending the note let it supply the missing
	// unit: `!chore 1 every` against a note reading "week groceries" parsed as
	// "every week groceries" and silently created a weekly chore nobody asked
	// for. A word that is not a unit makes an incomplete interval fail, which
	// is what the reply below is for.
	_, every, ok := ParseEvery(strings.TrimSpace(rest) + " " + intervalSentinel)
	if !ok {
		return Message{Text: "How often? Try !chore " + position + " every 2 weeks."}, nil
	}

	// The ordering argument that used to live here moved to PromoteItem, which
	// the screen calls too. One path, so the two views cannot disagree.
	c, ok, err := a.store.PromoteItem(ctx, personID, line.Item.ID, every)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(n), nil
	}
	return Message{Text: RenderDefined(c)}, nil
}

// snooze stops a chore asking for a while, without pretending it was done:
// `!snooze bins out`, `!snooze 1`, `!snooze bins out for 3 days`.
//
// The chore's clock keeps running while it is quiet, so when the time is up it
// is exactly as due as it was. That is the difference between this and `done`,
// and it is the whole reason this exists rather than people typing `done` at
// chores they have not done.
func (a *Applier) snooze(ctx context.Context, arg string, personID int64) (Message, error) {
	arg = strings.TrimSpace(arg)

	// "for 3 days" is peeled off the end so the rest can be a name with spaces
	// in it, which is what a chore name usually is.
	var how time.Duration
	said := ""

	// "now" is how the quiet ends early. It is the same write with a time in
	// the past rather than a second command, because a snooze that could not be
	// taken back would be a small permanent decision — and this product does
	// not have those.
	if subject, found := strings.CutSuffix(arg, " now"); found {
		arg, how, said = strings.TrimSpace(subject), 0, "asking again"
	} else if subject, rest, found := strings.Cut(arg, " for "); found {
		// "for a week" is what a person types; ParseEvery wants a number or
		// nothing where the article is. Dropping it here rather than widening
		// the pattern keeps one definition of what an interval looks like.
		rest = strings.TrimSpace(rest)
		for _, article := range []string{"a ", "an "} {
			rest = strings.TrimPrefix(rest, article)
		}
		_, parsed, ok := ParseEvery("every " + rest + " " + intervalSentinel)
		if !ok {
			return Message{Text: "How long? Try !snooze " + subject + " for 3 days."}, nil
		}
		arg, how = strings.TrimSpace(subject), parsed
		said = "for " + rest
	}

	active, err := a.store.ActiveChores(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	if len(active) == 0 {
		return Message{Text: "You have no chores, so there is nothing to put off."}, nil
	}
	if arg == "" {
		return Message{Text: "Which one, and for how long? Try !snooze " + active[0].Name + " for 3 days."}, nil
	}
	// No default, deliberately. A default is a decision made in advance for a
	// moment nobody can see, and the two obvious ones are both wrong somewhere:
	// a day is nothing for a fortnightly chore, and an interval is retiring by
	// another name. Asking costs one line and puts the decision where the
	// information is.
	if how == 0 && said == "" {
		return Message{Text: fmt.Sprintf("For how long? Try !snooze %s for 3 days, or a week, or a month.", arg)}, nil
	}

	target, instead, err := a.findChore(ctx, arg, personID, active)
	if err != nil {
		return Message{}, err
	}
	if target == nil {
		return instead, nil
	}

	if _, err := a.store.SnoozeChore(ctx, target.ID, personID, time.Now().Add(how)); err != nil {
		return Message{}, err
	}
	if how == 0 {
		return Message{Text: fmt.Sprintf("%s — %s.", target.Name, said)}, nil
	}
	return Message{Text: fmt.Sprintf("%s — quiet %s. It is no less done than it was.",
		target.Name, said)}, nil
}

// findChore turns what a person typed into one chore: a line number from the
// last listing, or a name.
//
// Both, because both are how you already refer to one. A number is what `?`
// just printed and a name is what you call it when you have not looked — and
// asking a person to look first, so they can quote a number back, is the tax
// this product exists to stop charging.
//
// When it cannot find one it returns nil and the message to send instead,
// which always names what you do have: an error that lists the alternatives is
// a recovery rather than a refusal.
func (a *Applier) findChore(ctx context.Context, arg string, personID int64, active []Chore) (*Chore, Message, error) {
	if n, err := strconv.Atoi(arg); err == nil {
		line, ok, err := a.store.LineAtPosition(ctx, personID, n)
		if err != nil {
			return nil, Message{}, err
		}
		if !ok {
			return nil, noSuchLine(n), nil
		}
		if line.Chore == nil {
			return nil, Message{Text: fmt.Sprintf("Line %d is a note, not a chore.", n)}, nil
		}
		return line.Chore, Message{}, nil
	}

	for i, c := range active {
		if strings.EqualFold(c.Name, arg) {
			return &active[i], Message{}, nil
		}
	}
	// A single unambiguous partial match counts. "bins" for "bins out" is what
	// a person types, and refusing it to be strict about a name they chose
	// themselves is pedantry with a cost.
	var found *Chore
	for i, c := range active {
		if strings.Contains(strings.ToLower(c.Name), strings.ToLower(arg)) {
			if found != nil {
				found = nil
				break
			}
			found = &active[i]
		}
	}
	if found != nil {
		return found, Message{}, nil
	}

	names := make([]string, 0, len(active))
	for _, c := range active {
		names = append(names, c.Name)
	}
	return nil, Message{Text: fmt.Sprintf("I don't have a chore called %q. You have: %s.",
		arg, strings.Join(names, ", "))}, nil
}

// did records a chore as done by naming it: `did bins out`, `did 2`.
//
// It exists because completing a chore from chat took two steps and a number
// held in your head — ask `?`, read the list, type `done 1`. The listing was
// load-bearing for the wrong reason: it was where the number came from, not
// where the decision was made.
//
// It records the same completion the tap does, through the same store call, so
// the two ways of saying it cannot mean different things.
func (a *Applier) did(ctx context.Context, arg string, personID int64) (Message, error) {
	arg = strings.TrimSpace(arg)

	active, err := a.store.ActiveChores(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	if len(active) == 0 {
		return Message{Text: "You have no chores, so there is nothing to have done."}, nil
	}
	if arg == "" {
		return Message{Text: "Which one? Try !did " + active[0].Name + "."}, nil
	}

	target, instead, err := a.findChore(ctx, arg, personID, active)
	if err != nil {
		return Message{}, err
	}
	if target == nil {
		return instead, nil
	}

	if err := a.store.RecordCompletion(ctx, target.ID, personID, "chat", time.Now()); err != nil {
		return Message{}, err
	}
	// The same varied reaction a tap earns, and varied for the same reason:
	// the same word every time stops being read inside a week.
	return Message{Text: fmt.Sprintf("%s %s.", Reactions[rand.Intn(len(Reactions))], target.Name)}, nil
}

// checkin asks how you are, or records the answer.
//
// `!mood` asks; `!mood low` answers in one go. Both, because the moment you
// know is not always the moment you were asked.
//
// It says nothing back about the person. "Noted" and the word you chose, and
// that is the whole reply — no "sorry to hear that", which is a stranger's
// sympathy from a program, and nothing at all about yesterday, which is the
// series this feature is forbidden from drawing.
func (a *Applier) checkin(ctx context.Context, arg string, personID int64) (Message, error) {
	arg = strings.TrimSpace(arg)
	if arg == "" {
		return CheckinQuestion(), nil
	}

	m, ok := ParseMood(arg)
	if !ok {
		return CheckinQuestion(), nil
	}
	if err := a.store.RecordCheckin(ctx, personID, m, "chat", time.Now()); err != nil {
		return Message{}, err
	}
	return Message{Text: "Noted — " + Words[m] + "."}, nil
}

// fix changes what a note says: `!fix 2 the boiler makes a noise on tuesdays`.
//
// This was listed as explicitly undecided for a long time, and the hesitation
// was worth having — a note is a record of what you said, and a record that
// can be rewritten is a record you cannot lean on. What settles it is the
// failure it fixes: a thought typed one-handed arrives half-written or
// autocorrected into something else, and until now the only remedy was to drop
// it and say it again, which costs the arrival time and the place in the pile.
//
// Only the words change. The arrival time, the state and the position stay,
// because those are the facts about the note and only the words were wrong.
func (a *Applier) fix(ctx context.Context, arg string, personID int64) (Message, error) {
	number, text, _ := strings.Cut(strings.TrimSpace(arg), " ")
	text = strings.TrimSpace(text)

	n, err := strconv.Atoi(number)
	if err != nil || n < 1 {
		return Message{Text: "Which line? Try !fix 2 the boiler makes a noise."}, nil
	}
	if text == "" {
		return Message{Text: fmt.Sprintf("What should line %d say? Try !fix %d the boiler makes a noise.", n, n)}, nil
	}

	line, ok, err := a.store.LineAtPosition(ctx, personID, n)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(n), nil
	}
	if line.Item == nil {
		return Message{Text: fmt.Sprintf("Line %d is a chore, not a note. Say it again with a new interval to change it.", n)}, nil
	}

	changed, err := a.store.Reword(ctx, personID, line.Item.ID, text)
	if err != nil {
		return Message{}, err
	}
	if !changed {
		return noSuchLine(n), nil
	}
	return Message{Text: "Fixed — " + text}, nil
}

// undoLast puts the most recently triaged note back in the pile.
//
// Which note that is comes from the store rather than from anything this
// applier remembers, so an undo typed in chat reverses a tap made on the
// screen: two views, one pile. Said twice it walks back another step, and
// nothing special happens on the second — an undo is a transition like any
// other, and so is undoing it.
//
// A promotion is the one case worth knowing about: undoing it returns the note
// and leaves the chore, because the chore is a separate thing that now exists.
// !retire is how that goes away, and saying so here is cheaper than guessing
// which chore a note became.
func (a *Applier) undoLast(ctx context.Context, personID int64) (Message, error) {
	it, ok, err := a.store.LastTriaged(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return Message{Text: "There is nothing to undo — nothing has left the pile."}, nil
	}
	if err := a.store.SetItemState(ctx, it.ID, ItemOpen, time.Now()); err != nil {
		return Message{}, err
	}
	return Message{Text: "Back in the pile — " + shorten(it.RawText)}, nil
}

// retire stops a chore coming back: `!retire bins out`, or `!retire 1` against
// whatever line 1 currently is.
//
// Two ways in, deliberately. A number is only reachable while a numbered
// surface is on screen, and the chore most worth retiring is the one that has
// been quietly nagging for weeks — which may not be on any list you can see. A
// name always works, and it is matched the way the unique index matches it, so
// the case you typed is not what decides.
//
// Retiring is not deleting. `active` goes false, the chore's history stays, and
// saying it again brings the same row back — which is what makes this safe to
// do on a whim, and why the reply says so.
func (a *Applier) retire(ctx context.Context, arg string, personID int64) (Message, error) {
	arg = strings.TrimSpace(arg)

	active, err := a.store.ActiveChores(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	if len(active) == 0 {
		return Message{Text: "You have no chores, so there is nothing to retire."}, nil
	}
	if arg == "" {
		return Message{Text: "Which one? Try !retire " + active[0].Name + "."}, nil
	}

	if n, err := strconv.Atoi(arg); err == nil {
		return a.retireLine(ctx, n, personID)
	}

	for _, c := range active {
		if strings.EqualFold(c.Name, arg) {
			return a.retireChore(ctx, c)
		}
	}

	// Saying what there is, rather than only what there is not: a name that
	// missed by a word is the common case, and the answer to it is the list.
	names := make([]string, 0, len(active))
	for _, c := range active {
		names = append(names, c.Name)
	}
	return Message{Text: fmt.Sprintf("I don't have a chore called %q. You have: %s.",
		arg, strings.Join(names, ", "))}, nil
}

func (a *Applier) retireLine(ctx context.Context, position int, personID int64) (Message, error) {
	line, ok, err := a.store.LineAtPosition(ctx, personID, position)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(position), nil
	}
	if line.Chore == nil {
		return Message{Text: fmt.Sprintf("Line %d is a note, not a chore.", position)}, nil
	}
	return a.retireChore(ctx, *line.Chore)
}

func (a *Applier) retireChore(ctx context.Context, c Chore) (Message, error) {
	if err := a.store.DeactivateChore(ctx, c.ID); err != nil {
		return Message{}, err
	}
	return Message{Text: fmt.Sprintf("%s — I will stop asking. Say it again to bring it back.", c.Name)}, nil
}

// numbered records a prompt whose lines are notes, so a typed position
// resolves back to the right one, and returns the message that prints them.
func (a *Applier) numbered(ctx context.Context, kind string, items []Item, more bool, personID int64, conversationID string) (Message, error) {
	if len(items) == 0 {
		// No prompt for an empty list: an empty numbered surface would still
		// become the newest one and would shadow a live nudge's numbering,
		// which is the shape phase 4 spent a round removing.
		return NotesMessage(items, more), nil
	}

	lines := make([]LineRef, 0, len(items))
	for i := range items {
		lines = append(lines, LineRef{ItemID: &items[i].ID})
	}
	id, err := a.store.RecordPromptLines(ctx, personID, conversationID, kind, time.Now(), nil, lines)
	if err != nil {
		return Message{}, err
	}
	a.pending = id
	return NotesMessage(items, more), nil
}

// noSuchLine is the one reply for a position nothing answers to. Shared so the
// three triage verbs and the chore path cannot drift into saying it differently
// — the wording is how you tell "I misread the number" from "that surface is
// gone", and two spellings would make that unreadable.
func noSuchLine(position int) Message {
	return Message{Text: fmt.Sprintf("I don't have a line %d.", position)}
}

// triage moves a note that a numbered line named. `done`, `keep` and `drop`
// differ only in the state they assert and what they say back.
//
// A position that turns out to name a chore is answered rather than silently
// ignored: `keep 2` on a chore is a real mistake, and a bot that does nothing
// looks broken in exactly the way that makes you stop trusting it.
func (a *Applier) triage(ctx context.Context, position int, personID int64, state ItemState, said string) (Message, error) {
	line, ok, err := a.store.LineAtPosition(ctx, personID, position)
	if err != nil {
		return Message{}, err
	}
	if !ok {
		return noSuchLine(position), nil
	}
	if line.Item == nil {
		return Message{Text: fmt.Sprintf("Line %d is a chore, not a note.", position)}, nil
	}
	if err := a.store.SetItemState(ctx, line.Item.ID, state, time.Now()); err != nil {
		return Message{}, err
	}
	return Message{Text: said + " " + shorten(line.Item.RawText)}, nil
}

// shorten trims a note to something that fits in a one-line acknowledgement.
// Quoting a whole paragraph back would bury the confirmation in the thing being
// confirmed.
func shorten(text string) string {
	const max = 60
	runes := []rune(text)
	if len(runes) <= max {
		return text
	}
	// Sliced by rune, not by byte: phase 2 crash-looped the pod on a name
	// containing Ⱥ because a byte slice cut a rune in half.
	return string(runes[:max]) + "…"
}

func (a *Applier) complete(ctx context.Context, in Intent, personID int64, conversationID string) (Message, error) {
	if in.Position > 0 {
		line, ok, err := a.store.LineAtPosition(ctx, personID, in.Position)
		if err != nil {
			return Message{}, err
		}
		if !ok {
			return noSuchLine(in.Position), nil
		}
		if line.Item != nil {
			return a.triage(ctx, in.Position, personID, ItemDone, "Done —")
		}
		c := *line.Chore
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return Message{}, err
		}
		return Message{Text: fmt.Sprintf("%s — next in %s.", c.Name, plural(c.EveryDays, "day"))}, nil
	}

	outstanding, err := a.store.OutstandingLines(ctx, personID)
	if err != nil {
		return Message{}, err
	}
	switch len(outstanding) {
	case 0:
		return Message{Text: "Nothing outstanding."}, nil
	case 1:
		c := outstanding[0]
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "ack", time.Now()); err != nil {
			return Message{}, err
		}
		return Message{Text: fmt.Sprintf("%s — next in %s.", c.Name, plural(c.EveryDays, "day"))}, nil
	default:
		// Never guess. Re-number and ask, so the reply can be a bare digit —
		// the same shape as IntentQuery, down to recording its own prompt.
		id, err := a.store.RecordPrompt(ctx, personID, conversationID, "query", time.Now(), nil, outstanding)
		if err != nil {
			return Message{}, err
		}
		a.pending = id
		m := ListMessage(outstanding)
		m.Text = "Which one?\n" + m.Text
		return m, nil
	}
}

// undo removes a chore the matcher created from what was meant as a note. It
// only reaches back undoWindow, so `nvm` long after the fact is not a
// destructive surprise.
func (a *Applier) undo(ctx context.Context, personID int64) (Message, error) {
	const q = `
		select id, name from chores
		 where person_id = $1 and active and created_at >= $2
		 order by created_at desc limit 1`

	var id int64
	var name string
	err := a.store.pool.QueryRow(ctx, q, personID, time.Now().Add(-undoWindow)).Scan(&id, &name)
	if err != nil {
		return Message{Text: "Nothing to undo."}, nil
	}
	if err := a.store.DeactivateChore(ctx, id); err != nil {
		return Message{}, err
	}
	return Message{Text: "Dropped " + name + ". It's still in your captures."}, nil
}

// isActionPayload reports whether the payload the transport stored alongside
// this text really came from an action webhook. ParseAction matches on text
// alone, and a person typing "!action 451 done:2 true" into the room produces
// text byte-identical to a genuine tap — the payload's "type" field is the
// only thing that tells them apart. Anything that fails to unmarshal or
// carries a different type is treated as not an action, which sends the
// message down the normal matcher: a thought is never rejected.
func isActionPayload(payload json.RawMessage) bool {
	var p struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &p); err != nil {
		return false
	}
	return p.Type == "action"
}

// isTap reports whether item is a genuine tap rather than a lookalike
// message — see isActionPayload. Text alone (ParseAction) is never enough on
// its own: someone can type "!action 5 done:1 true" into the room and
// produce text byte-identical to a real tap, so every caller that needs to
// tell the two apart must check both.
func isTap(item Item) bool {
	_, ok := ParseAction(item.RawText)
	return ok && isActionPayload(item.Payload)
}

// applyAction resolves a tap and applies it as a state assertion rather than a
// delta: "selected" means the completion should exist, "not selected" means it
// should not. Applying either twice lands in the same place, which is what
// makes a retried delivery harmless — the payload carries no event id, so a
// retry and a genuine second tap are indistinguishable.
//
// Every path here is silent. The boost is the receipt; a reply per tap would
// make the room unreadable.
func (a *Applier) applyAction(ctx context.Context, in ActionIntent, personID int64) error {
	// A mood is answered before anything is looked up: it is not about a chore,
	// so there is no prompt line to resolve and no position to mean anything.
	if in.Kind == "mood" {
		if !in.Selected {
			// Deselecting is not a mood. Nothing to record and nothing to
			// undo — the reading simply stands until the next one.
			return nil
		}
		m, ok := ParseMood(in.Mood)
		if !ok {
			return nil
		}
		return a.store.RecordCheckin(ctx, personID, m, "chat", time.Now())
	}

	prompt, ok, err := a.store.PromptByMessageID(ctx, personID, in.MessageID)
	if err != nil {
		return err
	}
	if !ok {
		// Someone else's prompt, or one we have no record of. Nothing to do,
		// and nothing to say: the capture is stored either way.
		return nil
	}

	c, ok, err := a.store.ChoreOnPrompt(ctx, prompt.ID, in.Position)
	if err != nil || !ok {
		return err
	}

	switch in.Kind {
	case "undefine":
		// DefinedMessage uses selection_mode "single", so deselecting the
		// button — or Campfire clearing the retained selection — delivers
		// "selected: false" for this same value. Only a selected tap is the
		// correction being asked for; anything else must be a no-op, the same
		// as an unselected "done" is for a completion.
		if !in.Selected {
			return nil
		}
		return a.store.DeactivateChore(ctx, c.ID)

	case "snooze":
		// "Not today", pressed. Tomorrow rather than a duration, because the
		// label is the duration: a button cannot ask how long, and the honest
		// answer is the one already written on it.
		//
		// Untapping clears it, which is the same shape as an unselected done
		// retracting a completion — "actually, ask me now" is the same write
		// with a time in the past rather than a second command.
		when := tomorrow(time.Now())
		if !in.Selected {
			when = time.Now().Add(-time.Minute)
		}
		_, err := a.store.SnoozeChore(ctx, c.ID, personID, when)
		return err

	case "done":
		if !in.Selected {
			_, err := a.store.RetractCompletion(ctx, c.ID, personID, prompt.ID, time.Now())
			return err
		}
		done, err := a.store.CompletedSince(ctx, c.ID, prompt.ID)
		if err != nil || done {
			return err
		}
		if err := a.store.RecordCompletion(ctx, c.ID, personID, "tap", time.Now()); err != nil {
			return err
		}
		a.react(ctx, prompt)
		return nil
	}
	return nil
}

// tomorrow is the start of the next day, locally. "Not today" means today's
// asking stops and tomorrow is a fresh question — not "in 24 hours", which
// would move the moment it asks a little later every time it was pressed.
func tomorrow(now time.Time) time.Time {
	y, m, d := now.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, now.Location()).AddDate(0, 0, 1)
}
