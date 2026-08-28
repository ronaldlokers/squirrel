package web

import (
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The way in.
//
// Squirrel's one screen of its own, and the first thing anybody ever sees of
// this product. Built the way /enough is built, which DESIGN.md describes as a
// treatment rather than a page: the mascot, a headline in the casual axis, a
// screen that is an absence. Nothing new is invented for it.
//
// Why a screen at all, when a straight redirect to Authentik would be one
// press fewer:
//
//   - This is a PWA. A cold launch from the home-screen icon that immediately
//     lurches to another domain is the awkward case, and worst on iOS.
//   - Signing out needs somewhere that does not bounce. Deleting the session
//     and landing on / would redirect to Authentik, which still has its own
//     session, and sign you straight back in.
//   - The unhappy states need a page anyway. As states of one screen they cost
//     nothing; as an afterthought they are three pages nobody drew.

// gateView is the whole of what this screen knows.
type gateView struct {
	// Said is the sentence under the mark, or empty for a first arrival.
	Said string
	// Button is what the control says. Three of the four states are still
	// trying to get in; one is trying again.
	Button string
	// Next is where to go afterwards, already checked.
	Next string
	V    string
	// Tilt and Light move with the day, the same two properties every other
	// screen hands the stylesheet.
	Tilt  int
	Light int
}

// saidAt is what each arrival says. An unknown value is a first arrival rather
// than an empty sentence or a panic: it comes out of a URL and anybody can
// type one.
//
// The refusal names no group. Which group an account lacks is a fact about the
// Authentik rather than about them, and saying it tells a stranger exactly
// what to ask for.
func saidAt(arrival string) (said, button string) {
	switch arrival {
	case "out":
		return "you are signed out", "LET ME IN"
	case "no":
		return "that account cannot use Squirrel", "LET ME IN"
	case "down":
		return "I cannot reach the way in just now", "TRY AGAIN"
	default:
		return "", "LET ME IN"
	}
}

func gateHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		said, button := saidAt(r.URL.Query().Get("said"))
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// Never cached. A signed-out screen served from a back button after
		// signing in is the two views disagreeing about who you are.
		w.Header().Set("Cache-Control", "no-store")
		if err := gatePage.ExecuteTemplate(w, "layout", gateView{
			Said:   said,
			Button: button,
			// Checked, not trusted. A value that arrives in a URL is a place a
			// stranger can type, and the same rule the timer's `from` uses.
			Next:  backTolerant(r.URL.Query().Get("next")),
			V:     stamp(),
			Tilt:  squirrel.Tilt(now()),
			Light: squirrel.Light(now()),
		}); err != nil {
			// Nothing more to fall back to: this is already the screen the
			// rest of the product falls back to.
			return
		}
	}
}
