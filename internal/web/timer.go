package web

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// The timer, on the screen.
//
// It sits on the chore card rather than on home, because you time the chore
// you are looking at. Running, it rides the lid on every screen — you started
// it in order to go and do something, and wandering to the pile should not
// lose it.
//
// The same row the chat writes, so !timer and this button cannot come to mean
// different things, and the one message at the end comes from the scheduler
// either way.
func timerHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		if err := r.ParseForm(); err != nil {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		// "leave me alone". It silences the exit ramp for the rest of the day
		// and leaves the timer exactly where it is — you did not say you were
		// stopping, you said you did not want to be asked.
		if r.FormValue("hush") != "" {
			if err := s.HushRamp(r.Context(), personID, now()); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
			return
		}

		// Stopping is its own answer and never an error: stopping something
		// that is not running is the state you asked for.
		if r.FormValue("stop") != "" {
			if err := s.StopTimer(r.Context(), personID); err != nil {
				fail(w, err)
				return
			}
			http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
			return
		}

		mins, err := strconv.Atoi(r.FormValue("minutes"))
		if err != nil || mins < 1 || mins > 180 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		label := strings.TrimSpace(r.FormValue("label"))
		if label == "" {
			label = "it"
		}
		if len(label) > choreNameLimit {
			label = label[:choreNameLimit]
		}

		if _, err := s.StartTimer(r.Context(), personID, label,
			time.Duration(mins)*time.Minute, now()); err != nil {
			fail(w, err)
			return
		}
		// The exit ramp, opted in on here and nowhere else. StartTimer clears
		// the flag, so this is an arming rather than a toggle: a timer with no
		// box ticked is a timer nothing watches, which is what every timer was
		// before this existed and what every timer the chat starts still is.
		if r.FormValue("ramp") != "" {
			if err := s.ArmRamp(r.Context(), personID, true); err != nil {
				fail(w, err)
				return
			}
		}
		http.Redirect(w, r, backFrom(r), http.StatusSeeOther)
	}
}

// backFrom returns to the screen the button was on, so stopping a timer from
// the pile does not land you on the chores.
func backFrom(r *http.Request) string {
	switch r.FormValue("from") {
	case "home":
		return "/"
	case "moods":
		return "/moods"
	}
	return "/"
}

// runningTimer is the strip in the lid: what you are doing and how long is
// left. The remaining time is the one number in this product that counts
// *down* — it is a fact about a thing you chose to start, not a total of what
// you have not done.
func runningTimer(s Store, opts Options, r *http.Request) *timerView {
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	t, found, err := s.CurrentTimer(r.Context(), personID)
	if err != nil || !found {
		return nil
	}
	left := t.Left(now())
	if left == 0 {
		// Up, but the scheduler has not said so yet. Showing 00:00 for up to a
		// minute is honest; showing nothing would look like it never ran.
		return &timerView{Label: t.Label, Left: "00:00"}
	}
	return &timerView{
		Label: t.Label,
		Left:  fmt.Sprintf("%02d:%02d", int(left.Minutes()), int(left.Seconds())%60),
	}
}

type timerView struct {
	Label string
	Left  string
}
