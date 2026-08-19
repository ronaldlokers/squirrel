package web

import "net/http"

// keptLimit caps the shelf. The cap is what makes "there is more" truthful,
// the same device the deck and search use.
const keptLimit = 30

// The shelf: the notes that were kept rather than done or dropped.
//
// It is not a third door. DESIGN.md allows exactly two, and their equality is
// the home screen's one statement — so this is reached from the ends of the
// pile instead, which is where you already are when there is nothing to triage
// and looking something up is the plausible next thing.
func keptHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := opts.person()
		if !ok {
			fail(w, errNoOwner)
			return
		}
		items, more, err := s.KeptItems(r.Context(), personID, keptLimit)
		if err != nil {
			fail(w, err)
			return
		}
		v := view{Here: "kept", More: more}
		for _, it := range items {
			v.Results = append(v.Results, toView(it))
		}
		render(w, "kept", v)
	}
}
