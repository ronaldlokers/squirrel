package squirrel

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// A photograph posted to the room arrives as its filename, and the note that
// results reads as nonsense a week later with nothing to explain it.
//
// Verified against production on 19 August 2026: an uncaptioned photo in the
// Campfire DM arrives with `body.plain` set to "IMG_5991.jpeg". The payload
// carries no attachment at all, so there is nothing else to keep. The note
// lands — capture is sacred and that is not in question — but its text hides
// its own content, which is the one thing this product exists to stop.
func TestAPhotographsNameIsRecognised(t *testing.T) {
	for _, name := range []string{
		"IMG_5991.jpeg",
		"IMG_5991.JPEG",
		"PXL_20260819_120000123.jpg",
		"photo.png",
		"scan.heic",
		"whiteboard.webp",
	} {
		require.True(t, LooksLikeAPhotographsName(name), "%q is a photograph's name", name)
	}
}

// And a thought is never mistaken for one. A false positive here is Squirrel
// explaining something that did not happen, which is worse than saying
// nothing: the note is fine and the explanation is the confusing part.
func TestAThoughtIsNotMistakenForAPhotograph(t *testing.T) {
	for _, said := range []string{
		"buy milk",
		"send the logo.png to the printer",
		"logo.png is the wrong one",
		"the boiler",
		"",
		"   ",
		".jpg",
		"~/pictures/img_1.jpg",
		`c:\photos\img_1.jpg`,
		"notes.txt",
		"budget.pdf",
	} {
		require.False(t, LooksLikeAPhotographsName(said), "%q is a thought", said)
	}
}

// What it says, and the order it says it in. The note is kept, and that comes
// first, because the one thing this must never read as is a refusal.
func TestItSaysTheNoteIsKeptBeforeItSaysWhatIsMissing(t *testing.T) {
	m := PhotographKeptByNameMessage("IMG_5991.jpeg")

	require.Contains(t, m.Text, "Kept")
	require.Contains(t, m.Text, "IMG_5991.jpeg", "it does not say which note it means")
	require.Contains(t, m.Text, "the camera on the pile",
		"it names what is missing without saying where the picture would go")
	require.Less(t, strings.Index(m.Text, "Kept"), strings.Index(m.Text, "not the photograph"),
		"it leads with what is missing rather than with what was kept")
	require.Empty(t, m.Actions, "nothing here is a thing to press")
}
