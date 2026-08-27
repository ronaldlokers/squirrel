package web

// The worked example a first-time visitor sees instead of an empty room.
//
// Two constraints, both easy to break by accident:
//
// Never stored. These are not turns and are never written — the first thing
// this product does must not be to write a record you did not make.
//
// Inert by construction. The verbs are spans, not forms, and no route exists
// that they could reach. Finding out what DONE does on somebody else's note is
// worse than not knowing.

type exampleTurn struct {
	Buddy bool
	Words string
	Card  *exampleCard
}

// exampleCard is a picture of a card. Acts are words, not controls.
type exampleCard struct {
	Name string
	Meta string
	Acts []string
}

// worked is the loop played once: a thing put down, what Squirrel says back,
// and a chore coming round. Three turns is the whole loop; a fourth would be
// teaching the product rather than showing it.
func worked() []exampleTurn {
	return []exampleTurn{
		{Words: "post the parcel back"},
		{
			Buddy: true,
			Words: "Down it goes.",
			Card: &exampleCard{
				Name: "post the parcel back",
				Meta: "just now · a task",
				Acts: []string{"DONE", "KEEP", "DROP"},
			},
		},
		{
			Buddy: true,
			Words: "The recycling is due today.",
			Card: &exampleCard{
				Name: "sort the recycling",
				Meta: "every thursday",
				Acts: []string{"DID IT", "NOT NOW"},
			},
		},
	}
}
