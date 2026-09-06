package web

import (
	"fmt"
	"log/slog"
	"net/http"
)

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
		if r.FormValue("stop") != "" {
			if err := s.StopTimer(r.Context(), personID); err != nil {
				fail(w, err)
				return
			}
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

const (
	shortestTimer = 1
	longestTimer  = 180
)

func runningTimer(s Store, opts Options, r *http.Request) *timerView {
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	t, found, err := s.CurrentTimer(r.Context(), personID)
	if err != nil {
		slog.Error("reading the timer", "error", err)
		return nil
	}
	if !found {
		return nil
	}
	left := t.Left(now())
	if left == 0 {
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
