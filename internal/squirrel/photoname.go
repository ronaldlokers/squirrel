package squirrel

import (
	"path"
	"regexp"
	"strings"
)

// A photograph posted to the room arrives as its filename and nothing else.
//
// Campfire's webhook carries no attachment: the payload has a body and the
// body has `plain`, and for an uncaptioned photo that plain text is
// "IMG_5991.jpeg". Verified against production on 19 August 2026. So the note
// is kept — capture is sacred and the row lands either way — but its text is
// the name of a file nobody can open from here, which is the product's own
// definition of a place a thought can hide.
//
// Fetching the bytes needs a change to the Campfire fork and is parked. What
// is not parked is saying so: a degradation the person cannot see is worse
// than the degradation itself, because next week the note reads as nonsense
// and there is nothing to explain it.

// photoName is a bare filename with a picture's extension and nothing else on
// the line. Deliberately narrow: no spaces, because a sentence that happens to
// end in a filename is a sentence, and a false positive here would be Squirrel
// explaining something that did not happen.
var photoName = regexp.MustCompile(`(?i)^\S+\.(jpe?g|png|heic|heif|gif|webp)$`)

// LooksLikeAPhotographsName reports whether this capture is a photograph that
// arrived as its own filename.
func LooksLikeAPhotographsName(text string) bool {
	trimmed := strings.TrimSpace(text)
	if !photoName.MatchString(trimmed) {
		return false
	}
	// A path is not a filename anybody's phone sends as a message body, and
	// treating one as a photograph would be guessing.
	return !strings.ContainsAny(trimmed, `/\`) && path.Ext(trimmed) != trimmed
}

// PhotographKeptByNameMessage is what Squirrel says about it.
//
// It says the note is kept first, because it is, and because the one thing
// this must never read as is a refusal. Then what is missing, then where the
// picture would have been kept — which is the screen, and which is the only
// part of this that asks anything of anyone.
func PhotographKeptByNameMessage(name string) Message {
	return Message{Text: "Kept — but only the name: " + name + ".\n" +
		"The room hands me what a photograph is called and not the photograph. " +
		"If the picture is the thing, the camera on the pile keeps it."}
}
