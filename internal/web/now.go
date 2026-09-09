package web

import (
	"net/http"

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
	return v
}

var offerKinds = map[squirrel.OfferKind]bool{
	squirrel.OfferChore:  true,
	squirrel.OfferTask:   true,
	squirrel.OfferAgain:  true,
	squirrel.OfferMoment: true,
	squirrel.OfferTimer:  true,
}
