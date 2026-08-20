package web

import (
	"net/http"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// How you have been, on the screen, and only when asked for by name.
//
// This table was unreadable by construction for the product's whole life: the
// store returned one reading and there was no function that could return more,
// which is a stronger guarantee than a rule somebody has to remember. That
// guarantee was given up deliberately on 20 August 2026, and what replaced it
// is narrower — there is one page and one command, both of which you have to
// go to, and nothing reads the readings on its own.
//
// So this file is the whole of the screen's half of the exception. Home shows
// today's answer and a link; the link is where the series lives; and the
// series is what you said with no arithmetic done to it.

func moodsHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		readings, err := s.CheckinsSince(r.Context(), personID, squirrel.MoodWindowStart(now()))
		if err != nil {
			fail(w, err)
			return
		}
		renderWith(w, r, s, opts, "moods", view{
			Here: "moods", Scrolling: true, Days: moodDays(readings, now()),
		})
	}
}

// moodDays groups the readings by the day they were said on.
//
// Grouped rather than listed, because two answers on one Tuesday is a Tuesday
// you checked in twice — not two facts about you. Newest day first, like every
// other list here.
//
// Nothing is counted, averaged or compared. A day with four readings renders
// four faces and says nothing about there being four.
func moodDays(readings []squirrel.Checkin, at time.Time) []moodDayView {
	days := []moodDayView{}
	for _, c := range readings {
		day := squirrel.MoodDay(c.SaidAt, at)
		if len(days) == 0 || days[len(days)-1].Day != day {
			days = append(days, moodDayView{Day: day})
		}
		here := &days[len(days)-1]
		here.Moods = append(here.Moods, faceView{
			Mood: string(c.Mood), Word: squirrel.Words[c.Mood],
		})
	}
	return days
}
