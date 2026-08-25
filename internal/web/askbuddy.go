package web

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Asking Buddy, in the conversation.
//
// The acorn opened a sheet over whatever you were looking at, and the sheet
// was its own conversation with its own window, its own scrollback and its own
// way of being closed. That made sense when the rest of the product was seven
// screens: Buddy had to bring a conversation with him because there was not
// one to join.
//
// There is one now. So the sheet is gone, the exchange is turns like every
// other exchange, and closing is not a thing you do to it — you stop talking,
// the same way you stop talking to anyone. What is kept is every part that
// does something: the four blockers, the steps a thing breaks into, the four
// proposals he must ask permission for, and saying that a reply landed badly.
//
// The window `remember`/`forget` maintained is not needed either: the record
// of what was said is the record of what was said. It is still written, because
// the next prompt is built from it.

// askBuddyChip is the way in, and it is on the live edge wherever that is.
//
// It has to be somewhere that always exists: the acorn was chrome and was
// therefore on every screen, and a chip that hangs off a card is a chip that
// disappears exactly when there is nothing to talk about.
func askBuddyChip() turnChip {
	return turnChip{Label: "ask Buddy", Action: "/buddy/ask"}
}

// findChip is the other one. Looking something up was a disclosure in the lid;
// the lid is a mark and nothing else now.
func findChip() turnChip {
	return turnChip{Label: "look something up", Action: "/find/ask"}
}

// findAskHandler is the search chip: it asks for words, and `/find` answers
// them. The field was a disclosure in the lid because it had to be reachable
// from seven screens; there is one screen.
func findAskHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "look something up"},
			askInWordsNamed("What are you looking for?", "/find", "q", "find it", nil),
		}), "/")
	}
}

// alwaysThere is the pair, for the live edge to carry.
func alwaysThere() []turnChip { return []turnChip{askBuddyChip(), findChip()} }

// askInWords is a question with a box under it, and the words to ask it.
//
// askForWords is the reword question and says so; this is the same shape with
// its own sentence, because "How should it read?" is the wrong question to ask
// somebody who pressed *ask Buddy*.
func askInWords(question, action, does string, fields map[string]string) squirrel.Turn {
	return askInWordsNamed(question, action, "said", does, fields)
}

// askInWordsNamed is the same, for a route whose field is called something
// else. Search has always taken `q`, and renaming it would break the one URL
// in this product a person might have typed.
func askInWordsNamed(question, action, field, does string, fields map[string]string) squirrel.Turn {
	body, err := json.Marshal(drawn{Say: &sayView{
		Action: action, Field: field, Label: question, Fields: fields, Do: does,
	}})
	if err != nil {
		slog.Error("drawing the question", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: question}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: question, Shown: body}
}

// coachReply is what Buddy said, as a turn.
//
// The four blockers ride on it when there is nothing better to say — a chip is
// the sentence you did not have to type, and somebody at the moment of least
// capacity should not have to compose anything to be helped. "That went badly"
// rides on any reply the model wrote, and on none that it did not: there is no
// point telling us a fixed sentence landed badly when the fixed sentence is
// the same one every time.
func coachReply(words string, askWhich, fromModel bool, p *Proposal, step *stepView) squirrel.Turn {
	return coachReplyCosting(words, "", askWhich, fromModel, p, step)
}

// coachReplyCosting is the same with what this has cost written on it.
//
// The sheet carried the spend in its own lid, and the rule it was protecting
// is that a running cost must never be a number you meet before you have asked
// for anything. The sheet is gone and the home screen is the only chrome left,
// so the figure moves to the one place that still satisfies the rule: the
// reply itself. You have asked by the time you can read it, and it is not on
// screen at any other moment.
func coachReplyCosting(words, cost string, askWhich, fromModel bool, p *Proposal, step *stepView) squirrel.Turn {
	sh := drawn{Cost: cost}
	if askWhich {
		for _, b := range squirrel.Blockers {
			sh.Chips = append(sh.Chips, turnChip{
				Label: squirrel.BlockerWords[b], Action: "/buddy/say",
				Fields: map[string]string{"why": string(b)},
			})
		}
	}
	if step != nil {
		sh.Cards = append(sh.Cards, stepCard(step))
	}
	if p != nil {
		sh.Cards = append(sh.Cards, proposalCard(p))
	}
	if fromModel {
		sh.Chips = append(sh.Chips, turnChip{Label: "that went badly", Action: "/buddy/badly"})
	}
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing what Buddy said", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}
}

// stepCard is where you are in a breakdown.
//
// Kept whole, because coming back an hour later and finding the step you were
// on is the entire reason the sequence is stored rather than held in a reply.
func stepCard(st *stepView) cardView {
	c := cardView{Title: st.Label, Meta: st.Body}
	if st.Last {
		c.Meta = strings.TrimSpace(st.Body + " — the last one")
	}
	return cardView{
		Title: c.Title, Meta: c.Meta,
		Acts: []actView{
			{Label: "done", Action: "/steps", Style: "did", Fields: map[string]string{
				"act": "done", "id": strconv.FormatInt(st.ID, 10), "from": "/",
			}},
			{Label: "throw it away", Action: "/steps", Style: "why", Fields: map[string]string{
				"act": "clear", "from": "/",
			}},
		},
	}
}

// proposalCard is a thing Buddy wants permission for, as one press.
//
// Stored nowhere, exactly as it was: it travels in the form that renders it,
// so a proposal in scrollback has lost its button by the live edge rule, and
// nothing is applied by anything except a press. That is the same guarantee
// the sheet gave, arrived at by the rule the whole conversation already runs
// on rather than by a page that happens not to survive a reload.
func proposalCard(p *Proposal) cardView {
	fields := map[string]string{"do": p.Do, "text": p.Text}
	if p.At != "" {
		fields["at"] = p.At
	}
	if p.Every != "" {
		fields["every"] = p.Every
	}
	if p.RefID != 0 {
		fields["id"] = strconv.FormatInt(p.RefID, 10)
	}
	// KEEP IT and nothing beside it. "Never mind" was a link back to the page,
	// and there is no page: nothing was written, so not pressing is already
	// how you decline — and the live edge takes the button away the moment you
	// say anything else.
	return cardView{
		Title: p.Said, Meta: p.Text,
		Acts: []actView{{Label: "KEEP IT", Action: "/buddy/do", Style: "did", Fields: fields}},
	}
}

// offerHint is what you would be handed right now, said rather than painted.
//
// The sheet drew the offer at the top so a conversation had something to be
// about. In the thread the conversation already has whatever is above it, so
// this is only asked for when Buddy is being opened cold — and it costs
// nothing, because the picker is six rules and no model.
func offerHint(s Store, opts Options, r *http.Request) string {
	// Through offerFor, which is the path that goes via the coach's cache. It
	// may consult a decision that was already paid for and may never cause
	// one: asking has to stay free, and the first version of this test only
	// checked the conversational seam while the picker's seam paid for a tool
	// loop on every press.
	o := offerFor(s, opts, r, true, false)
	if o == nil {
		return ""
	}
	return o.Text
}
