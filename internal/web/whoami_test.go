package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The rail is two shapes from one markup — a column on a desktop and the sheet
// behind the room control on a phone — so this is both surfaces at once.
func TestTheRailSaysWhoYouAreAndTheWayOut(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `<span class="whoname">Ronald Lokers</span>`,
		"the rail does not say who it is talking to")
	require.Contains(t, body, `<form method="post" action="/auth/out">`,
		"there is no way out of the product")
	require.NotContains(t, body, `<a class="signout" href="/auth/out"`,
		"signing out is a GET, which is broken and a cross-site press away from being a way to sign somebody out")
}

// A person the gate never learned a name for still gets a face. The letter is
// drawn rather than fetched, so there is nothing to go missing.
func TestWithoutAPictureTheFaceIsDrawn(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `<span class="youface mono" aria-hidden="true">R</span>`,
		"no picture and no monogram either, so the name stands beside nothing")
	require.NotContains(t, body, `src="/me/face"`,
		"a picture is fetched that the store says is not there")
}

func TestWithAPictureTheFaceIsServedFromHere(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers", whoFace: []byte("not really a png")}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	require.Contains(t, body, `src="/me/face"`, "the picture is not shown")
	require.NotContains(t, body, "https://", "a face is fetched from somewhere that is not this origin")
}

// Your turns carry a face the way Buddy's do, and only where a run starts.
func TestYourTurnsCarryYourFace(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	f.turns = []squirrel.Turn{
		{ID: 1, Who: squirrel.SpeakerYou, Words: "the tasks"},
		{ID: 2, Who: squirrel.SpeakerBuddy, Words: "Two come back round."},
	}
	body := mounted(t, f).call(t, "GET", "/", nil).Body.String()

	// Inside the turn, not merely somewhere on the page: the rail draws a
	// youface too, so a bare substring passes with the turn's face deleted.
	yours := body[strings.Index(body, `class="turn fromyou`):]
	yours = yours[:strings.Index(yours, `class="turn frombuddy`)]
	require.Contains(t, yours, `class="youface`, "your own turns have no face")
	require.Contains(t, body, `class="buddyface"`, "his did not survive yours arriving")
}

func TestYourFaceIsNotServedToNobody(t *testing.T) {
	f := &fakeStore{whoFace: []byte("bytes")}
	w := mounted(t, f).call(t, "GET", "/me/face", nil)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "image/png", w.Header().Get("Content-Type"))
	require.Contains(t, w.Header().Get("Cache-Control"), "private",
		"your face is cacheable by something that is not your browser")

	none := mounted(t, &fakeStore{}).call(t, "GET", "/me/face", nil)
	require.Equal(t, http.StatusNotFound, none.Code,
		"a person with no picture is served something rather than nothing")
}

// answering is a client that says whatever a test wants it to.
type answering struct {
	kind string
	body string
	code int
}

func (a answering) Do(*http.Request) (*http.Response, error) {
	code := a.code
	if code == 0 {
		code = 200
	}
	res := httptest.NewRecorder()
	res.Header().Set("Content-Type", a.kind)
	// Before the body: writing first sends a 200 and the status under test
	// never happens.
	res.WriteHeader(code)
	_, _ = res.WriteString(a.body)
	return res.Result(), nil
}

// Everything the gate hands over is somebody else's string, and a picture is
// the one that arrives as bytes from a URL the provider chose.
func TestAPictureIsOnlyKeptWhenItIsOne(t *testing.T) {
	for _, c := range []struct {
		what string
		from string
		with answering
		keep bool
	}{
		{"an ordinary avatar", "https://authentik.example/face.png", answering{kind: "image/png", body: "PNG"}, true},
		{"no claim at all", "", answering{kind: "image/png", body: "PNG"}, false},
		{"plain http", "http://authentik.example/face.png", answering{kind: "image/png", body: "PNG"}, false},
		{"a page, not a picture", "https://authentik.example/face", answering{kind: "text/html", body: "<html>"}, false},
		{"a refusal", "https://authentik.example/face.png", answering{kind: "image/png", body: "no", code: 404}, false},
		{"nothing at all", "https://authentik.example/face.png", answering{kind: "image/png", body: ""}, false},
		{"more than fits", "https://authentik.example/face.png", answering{kind: "image/png", body: strings.Repeat("x", faceLimit+1)}, false},
	} {
		t.Run(c.what, func(t *testing.T) {
			face, kind := fetchFace(t.Context(), Options{Fetch: c.with}, c.from)
			if c.keep {
				require.NotEmpty(t, face, "a picture that is fine was thrown away")
				require.Equal(t, "image/png", kind)
				return
			}
			require.Empty(t, face, "kept something that is not a picture worth keeping")
			require.Empty(t, kind)
		})
	}
}
