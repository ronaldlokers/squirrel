package web

import (
	"log/slog"
	"net/http"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func meHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		v := view{Here: "you"}
		v.Weeks, v.MoodsSays = howYouHaveBeen(r, s, personID)
		v.Known, v.KnownSays = whatIsKnown(r, s, personID)
		renderWith(w, r, s, opts, "me", v)
	}
}

func howYouHaveBeen(r *http.Request, s Store, personID int64) ([]moodWeekView, string) {
	readings, err := s.CheckinsSince(r.Context(), personID, squirrel.MoodCalendarStart(now()))
	if err != nil {
		slog.Error("reading how you have been", "error", err)
		return nil, "I cannot reach those just now."
	}
	weeks := moodWeeks(readings, now())
	if len(weeks) == 0 {
		return nil, "You have not said how you are lately."
	}
	return weeks, ""
}

func whatIsKnown(r *http.Request, s Store, personID int64) ([]string, string) {
	known, err := s.Knowing(r.Context(), personID)
	if err != nil {
		slog.Error("reading what is known", "error", err)
		return nil, "I cannot reach that just now."
	}
	if len(known) == 0 {
		return nil, "Nothing yet. I read back what we have said about once a week, " +
			"and write down what it seems to show."
	}
	return known, "This is what our conversations seem to show. I could be wrong about any of it."
}

func meForgetHandler(s Store) http.HandlerFunc {
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
		http.Redirect(w, r, "/me", http.StatusSeeOther)
	}
}
