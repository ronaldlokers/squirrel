package web

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func offerFor(s Store, r *http.Request, anyway bool) *offerView {
	personID, ok := personOf(r)
	if !ok {
		return nil
	}
	o, found, err := s.PickNow(r.Context(), personID, now(), anyway)
	if err != nil || !found {
		return nil
	}
	v := &offerView{
		Kind:    string(o.Kind),
		RefID:   o.RefID,
		Text:    o.Text,
		Because: o.Because,
	}
	v.Running = o.Kind == squirrel.OfferTimer
	return v
}

var offerKinds = map[squirrel.OfferKind]bool{
	squirrel.OfferChore:  true,
	squirrel.OfferTask:   true,
	squirrel.OfferAgain:  true,
	squirrel.OfferMoment: true,
	squirrel.OfferTimer:  true,
}

func startFromOffer(s Store, r *http.Request, personID int64) error {
	mins, err := strconv.Atoi(r.FormValue("minutes"))
	if err != nil || mins < 1 || mins > 180 {
		return nil
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
		return err
	}

	kind := squirrel.OfferKind(r.FormValue("kind"))
	refID, _ := strconv.ParseInt(r.FormValue("id"), 10, 64)
	return s.RecordAnswer(r.Context(), personID, kind, refID, squirrel.AnswerStarted, now())
}
