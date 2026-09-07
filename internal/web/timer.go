package web

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
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

type rampView struct {
	Label string
}

const (
	quietFrom  = 22
	quietUntil = 6
)

func quietHours(at time.Time) bool {
	h := at.Hour()
	return h >= quietFrom || h < quietUntil
}

func armRampIfTicked(r *http.Request, s Store, personID int64) error {
	if r.FormValue("ramp") != "1" {
		return nil
	}
	return s.ArmRamp(r.Context(), personID, true)
}

func rampFor(s Store, r *http.Request, personID int64, at time.Time) *rampView {
	if quietHours(at) {
		return nil
	}
	if s.Capacity(r.Context(), personID, at) == squirrel.CapacityLow {
		return nil
	}
	t, found, err := s.RampDue(r.Context(), personID, at)
	if err != nil {
		slog.Error("reading the exit ramp", "error", err)
		return nil
	}
	if !found {
		return nil
	}
	if err := s.RampSaid(r.Context(), personID, at); err != nil {
		slog.Error("marking the exit ramp said", "error", err)
	}
	return &rampView{Label: t.Label}
}
