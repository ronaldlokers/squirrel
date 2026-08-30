package web

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// What Squirrel thinks it knows about you, shown to you.
//
// This is the only thing in the product that holds an opinion about somebody
// rather than something they said, and it is written by a model reading a
// record you cannot see it read. A product that quietly builds a picture of
// you that you cannot read is not this product — so it is one press away, in
// the model's own words rather than a summary of them, and one press to throw
// away.
//
// Not a place you go. It is a thing you check when you wonder, so it hangs off
// Buddy's own turn, which is where wondering about Buddy happens.

// knowingChip hangs off Buddy's reply. It says what it is rather than naming a
// feature: "what you know about me" is the question somebody actually has.
func knowingChip() turnChip {
	return turnChip{Label: "what do you know about me", Action: "/knowing"}
}

// knowingHandler draws it.
//
// One card per observation, so each has its own shape on the screen and none
// of them reads as a paragraph of assessment. No date on them: when Squirrel
// worked something out is not a fact you can act on, and a timestamp beside an
// opinion about you invites reading it as a file.
func knowingHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		known, err := s.Knowing(r.Context(), personID)
		if err != nil {
			answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
				{Who: squirrel.SpeakerYou, Words: "what do you know about me"},
				{Who: squirrel.SpeakerBuddy, Words: "I cannot reach that just now."},
			}), backToTheRoom(r))
			return
		}

		said := []squirrel.Turn{{Who: squirrel.SpeakerYou, Words: "what do you know about me"}}
		if len(known) == 0 {
			// Nothing yet is the ordinary state for a first week, and it says
			// what would change that rather than apologising for it.
			said = append(said, squirrel.Turn{
				Who: squirrel.SpeakerBuddy,
				Words: "Nothing yet. I read back what we have said about once a week, " +
					"and write down what it seems to show.",
			})
			answerWith(w, r, keepSaid(r.Context(), s, personID, said), backToTheRoom(r))
			return
		}

		sh := drawn{Place: "what I know about you"}
		for _, one := range known {
			sh.Cards = append(sh.Cards, cardView{Title: one})
		}
		sh.Chips = []turnChip{{Label: "forget all of it", Action: "/knowing/forget"}}

		said = append(said, sayWithCards(
			"This is what our conversations seem to show. I could be wrong about any of it.", sh))
		answerWith(w, r, keepSaid(r.Context(), s, personID, said), backToTheRoom(r))
	}
}

// forgetKnowingHandler throws it away.
//
// One press, no confirmation, the same shape every other reversal here has.
// What it costs is a week, and the answer says so — a control whose
// consequence is invisible is a control you press once and then wonder about.
func forgetKnowingHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := s.ForgetKnowing(r.Context(), personID); err != nil {
			fail(w, err)
			return
		}
		answerWith(w, r, keepSaid(r.Context(), s, personID, []squirrel.Turn{
			{Who: squirrel.SpeakerYou, Words: "forget all of it"},
			{Who: squirrel.SpeakerBuddy,
				Words: "Forgotten. I will start again from what we say from here."},
		}), backToTheRoom(r))
	}
}

// sayWithCards is a sentence with things drawn under it. sayWithChips is the
// same shape for a turn that only carries a way out.
func sayWithCards(words string, sh drawn) squirrel.Turn {
	body, err := json.Marshal(sh)
	if err != nil {
		slog.Error("drawing what is known", "error", err)
		return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words}
	}
	return squirrel.Turn{Who: squirrel.SpeakerBuddy, Words: words, Shown: body}
}
