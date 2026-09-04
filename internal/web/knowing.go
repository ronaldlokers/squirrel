package web

import (
	"encoding/json"
	"log/slog"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// What Squirrel thinks it knows about you, and the shape a sentence with
// things drawn under it takes.
//
// The reading itself moved to /me on 3 September 2026: it is the only thing in
// the product that holds an opinion about somebody rather than about something
// they said, so it belongs on the page about you rather than behind a press
// that answered in a room.

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
