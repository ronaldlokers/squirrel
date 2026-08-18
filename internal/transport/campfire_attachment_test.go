package transport_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/transport"
)

// A photo of the boiler's error code, or a voice memo in the car, is a thought
// like any other — and until now it was the one kind that could be lost. The
// message arrives with an empty body, lands as a row with no text, and every
// surface filters it out: not in the pile, not in the evening list, not
// findable. It is there in the database and nowhere a person can see.
//
// Capture is sacred. Something has to stand in for it.
func TestAnAttachmentWithNoWordsIsStillANote(t *testing.T) {
	for _, tc := range []struct {
		name    string
		payload string
		want    string
	}{
		{
			name: "named in the message",
			payload: `{
			  "user": {"id": 1}, "room": {"id": 7},
			  "message": {"id": 42, "body": {"plain": ""},
			              "attachment": {"filename": "boiler-error.jpeg", "content_type": "image/jpeg"}}
			}`,
			want: "📎 boiler-error.jpeg",
		},
		{
			name: "named beside the message",
			payload: `{
			  "user": {"id": 1}, "room": {"id": 7},
			  "message": {"id": 42, "body": {"plain": "  "}},
			  "attachment": {"name": "voice-memo.m4a"}
			}`,
			want: "📎 voice-memo.m4a",
		},
		{
			name: "one of several",
			payload: `{
			  "user": {"id": 1}, "room": {"id": 7},
			  "message": {"id": 42, "body": {"plain": ""},
			              "attachments": [{"filename": "meter.png"}, {"filename": "second.png"}]}
			}`,
			want: "📎 meter.png",
		},
		{
			name: "there, but nameless",
			payload: `{
			  "user": {"id": 1}, "room": {"id": 7},
			  "message": {"id": 42, "body": {"plain": ""},
			              "attachment": {"url": "https://campfire.example/attachments/9"}}
			}`,
			want: "📎 an attachment",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := transport.CaptureFrom([]byte(tc.payload), at)
			require.Equal(t, tc.want, c.Text)
		})
	}
}

// Words win. An attachment with a caption is the caption — the placeholder is
// for the case where there is nothing else to show, not a label stuck on
// everything that arrives with a file.
func TestACaptionedAttachmentKeepsItsWords(t *testing.T) {
	c := transport.CaptureFrom([]byte(`{
	  "user": {"id": 1}, "room": {"id": 7},
	  "message": {"id": 42, "body": {"plain": "the boiler error code"},
	              "attachment": {"filename": "boiler-error.jpeg"}}
	}`), at)

	require.Equal(t, "the boiler error code", c.Text)
}

// An empty message with no attachment stays empty. Campfire sends bodiless
// events for its own reasons, and turning every one of them into a note that
// says "an attachment" would fill the pile with things nobody sent.
func TestAnEmptyMessageWithNoAttachmentStaysEmpty(t *testing.T) {
	c := transport.CaptureFrom([]byte(`{
	  "user": {"id": 1}, "room": {"id": 7},
	  "message": {"id": 42, "body": {"plain": ""}}
	}`), at)

	require.Empty(t, c.Text)
}

// The evidence has to be an attachment, not the word appearing anywhere in the
// payload — a message quoting the word "attachment" with an empty body is
// still an empty body.
func TestTheEvidenceIsAnAttachmentNotTheWord(t *testing.T) {
	c := transport.CaptureFrom([]byte(`{
	  "user": {"id": 1}, "room": {"id": 7, "name": "attachments"},
	  "message": {"id": 42, "body": {"plain": ""}}
	}`), at)

	require.Empty(t, c.Text)
}
