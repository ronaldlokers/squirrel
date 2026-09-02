//go:build browser

package web

// The live edge's browser tests went with the four rooms on 2 September 2026.
//
// They pinned that only the edge carried controls and that acting on the list
// changed the list — both facts about a room drawing its own list below the
// conversation. A rack is that list, it re-reads on every load, and every strip
// in it carries its own answers, which is a different shape with the same
// promise. The board's own tests hold it: TestBrowserTheBoardsKeysFollowFocus,
// TestAnsweringAStripMovesItAndComesBackToTheBoard, and the strike-and-hold
// test that pins the undo's home.
