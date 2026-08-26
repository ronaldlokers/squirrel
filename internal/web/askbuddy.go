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
// The exchange is turns like any other, so closing is not a thing you do to it
// — you stop talking. What is kept from the sheet that preceded it is every
// part that does something: the four blockers, the steps a thing breaks into,
// the four proposals he must ask permission for, and saying a reply landed
// badly.
//
// remember/forget still writes the rolling window, because the next prompt is
// built from it.

// askBuddyChip is the way in, on the live edge wherever that is. It has to be
// somewhere that always exists: a chip hanging off a card disappears exactly when
// there is nothing to talk about.
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
		personID, ok := personOf(r)
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

// askInWords is a question with a box under it. Its own sentence rather than
// askForWords', because "How should it read?" is the wrong question to ask
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

// coachReply is what Buddy said, as a turn. The four blockers ride on it when
// there is nothing better to say. "That went badly" rides on any reply the model
// wrote and on none that it did not.
func coachReply(words string, askWhich, fromModel bool, p *Proposal, step *stepView) squirrel.Turn {
	return coachReplyCosting(words, "", askWhich, fromModel, p, step)
}

// coachReplyCosting is the same with what this has cost written on it. A running
// cost must never be a number you meet before you have asked for anything, and by
// the time you can read this you have asked.
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
		// And the way to see what is behind the answer. Only on a reply a
		// model wrote: what Squirrel knows about you shapes those and nothing
		// else, so beside a fixed sentence from the core it would be pointing
		// at something that had no part in it.
		sh.Chips = append(sh.Chips, knowingChip())
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

// proposalCard is a thing Buddy wants permission for, as one press. Stored
// nowhere: it travels in the form that renders it, so a proposal in scrollback
// has lost its button by the live edge rule.
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

// offerHint is what you would be handed right now, asked for only when Buddy is
// opened cold — the conversation already has whatever is above it. It costs
// nothing: the picker is six rules and no model.
func offerHint(s Store, opts Options, r *http.Request) string {
	// Through offerFor, the path that goes via the coach's cache: it may consult a
	// decision already paid for and may never cause one. Asking has to stay free.
	o := offerFor(s, opts, r, true, false)
	if o == nil {
		return ""
	}
	return o.Text
}
