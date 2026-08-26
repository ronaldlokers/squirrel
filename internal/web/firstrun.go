package web

// The first thing anybody ever sees, and the one screen that has to teach
// itself.
//
// A conversation with nothing in it is an empty room with a box in it saying
// *put it down here*. That asks a stranger to trust the product with something
// before showing them what it will do with it, and the thing being asked for —
// the contents of somebody's head — is the one thing they have most reason not
// to hand over first.
//
// So Buddy plays the loop through once, dimmed, and then rules a line under it
// and hands over. Seeing your own bubble beside Buddy's unbubbled words teaches
// the grammar of the screen in one look, which no sentence about it can do.
//
// Two rules hold this down:
//
// It is never stored. These are not turns, they are not written, and they are
// gone the moment there is one real turn. The first thing this product does
// must not be to write a false memory — a record you did not make, in a record
// whose whole value is that you made all of it.
//
// It is inert by construction. The verbs here are spans, not forms, and there
// is no route they could reach if they were pressed. A worked example wired to
// real controls is a trap: the one thing worse than not knowing what DONE does
// is finding out on somebody else's note.

// exampleTurn is one line of the worked example.
type exampleTurn struct {
	Buddy bool
	Words string
	Card  *exampleCard
}

// exampleCard is a card drawn as a picture of a card. Acts are words, not
// controls.
type exampleCard struct {
	Name string
	Meta string
	Acts []string
}

// worked is the loop, played once: a thing put down, what Squirrel says back,
// and a chore being offered.
//
// Three turns, because three is the whole loop — you say something, it is kept
// and named, and later it comes back to you. A fourth would be teaching the
// product rather than showing it.
//
// The words are ordinary and small on purpose. `post the parcel back` is the
// kind of thing this is for; an example built out of impressive things would be
// a demonstration of the product rather than a picture of your Tuesday.
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
