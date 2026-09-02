package web

// The edge went with the four rooms on 2 September 2026.
//
// It was the current state of a room, drawn under the conversation and askable
// on its own so the script could refresh it without taking the page. The board
// is that by construction: every rack re-reads on load, and there is no
// scrollback for a live thing to sit at the bottom of. Buddy's room keeps a
// live edge for its own turns, which the thread's tests cover.
//
// What was here: an edge could be asked for on its own, what it answered was
// current rather than what the room drew when you arrived, and every form it
// drew carried its room. The first two are what a rack is; the third is
// covered by TestEveryFormATurnDrawsCarriesItsRoom.
