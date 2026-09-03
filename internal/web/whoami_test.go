package web

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The rail went with the rooms on 3 September 2026 and what it held is a page.
func TestTheSettingsSayWhoYouAreAndTheWayOut(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	body := mounted(t, f).call(t, "GET", "/me", nil).Body.String()

	require.Contains(t, body, `<span class="whoname">Ronald Lokers</span>`,
		"the page does not say who it is talking to")
	require.Contains(t, body, `<form method="post" action="/auth/out">`,
		"there is no way out of the product")
	require.NotContains(t, body, `<a class="signout" href="/auth/out"`,
		"signing out is a GET, which is broken and a cross-site press away from being a way to sign somebody out")
}

// A person the gate never learned a name for still gets a face. The letter is
// drawn rather than fetched, so there is nothing to go missing.
func TestWithoutAPictureTheFaceIsDrawn(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers"}
	body := mounted(t, f).call(t, "GET", "/me", nil).Body.String()

	require.Contains(t, body, `<span class="youface mono" aria-hidden="true">R</span>`,
		"no picture and no monogram either, so the name stands beside nothing")
	require.NotContains(t, body, `src="/me/face"`,
		"a picture is fetched that the store says is not there")
}

func TestWithAPictureTheFaceIsServedFromHere(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers", whoFace: []byte("not really a png")}
	body := mounted(t, f).call(t, "GET", "/me", nil).Body.String()

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
	body := mounted(t, f).call(t, "GET", "/r/everything", nil).Body.String()

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

	require.Equal(t, "nosniff", w.Header().Get("X-Content-Type-Options"),
		"a mislabelled body can be sniffed into something that runs")

	// A row written before the list existed, or by a version that trusted the
	// remote server. What a browser does with a document is decided here.
	odd := &fakeStore{whoFace: []byte("<svg/>"), whoFaceType: "image/svg+xml"}
	svg := mounted(t, odd).call(t, "GET", "/me/face", nil)
	require.Equal(t, "application/octet-stream", svg.Header().Get("Content-Type"),
		"a stored svg is served as one, from this origin, under somebody's avatar")

	none := mounted(t, &fakeStore{}).call(t, "GET", "/me/face", nil)
	require.Equal(t, http.StatusNotFound, none.Code,
		"a person with no picture is served something rather than nothing")
}

// aRealPNG is eight bytes of signature and nothing else: enough for
// http.DetectContentType to agree it is a PNG, which is the point.
const aRealPNG = "\x89PNG\r\n\x1a\n"

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
		{"an ordinary avatar", "https://authentik.example/face.png", answering{kind: "image/png", body: aRealPNG}, true},
		{"no claim at all", "", answering{kind: "image/png", body: aRealPNG}, false},
		{"plain http", "http://authentik.example/face.png", answering{kind: "image/png", body: aRealPNG}, false},
		{"a page, not a picture", "https://authentik.example/face", answering{kind: "text/html", body: "<html>"}, false},
		{"a refusal", "https://authentik.example/face.png", answering{kind: "image/png", body: aRealPNG, code: 404}, false},
		{"nothing at all", "https://authentik.example/face.png", answering{kind: "image/png", body: ""}, false},
		{"more than fits", "https://authentik.example/face.png", answering{kind: "image/png", body: aRealPNG + strings.Repeat("x", faceLimit)}, false},
		// An SVG is a document that runs script. Served from this origin under
		// somebody's avatar it is stored cross-site scripting, so it never
		// becomes bytes we hold.
		// The bytes and the header agree, and it is still refused: four formats
		// cover every avatar anyone serves, and a shorter list is a smaller
		// surface for whatever the next image decoder bug turns out to be.
		{"a format nothing needs", "https://authentik.example/face.bmp", answering{kind: "image/bmp", body: "BM" + strings.Repeat("\x00", 30)}, false},
		{"a drawing that runs", "https://authentik.example/face.svg", answering{kind: "image/svg+xml", body: `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`}, false},
		{"an svg wearing a png label", "https://authentik.example/face.png", answering{kind: "image/png", body: `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`}, false},
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

// The picture URL is somebody else's, and a signed claim proves only that
// Authentik said it. Everything that is not the open internet is refused.
func TestAPictureNeverComesFromInsideTheHouse(t *testing.T) {
	for _, c := range []struct {
		what string
		ip   string
		open bool
	}{
		{"an ordinary host", "93.184.216.34", true},
		{"an ordinary v6 host", "2606:2800:220:1:248:1893:25c8:1946", true},
		{"localhost", "127.0.0.1", false},
		{"localhost, in v6", "::1", false},
		{"this machine's other name", "0.0.0.0", false},
		{"the cloud metadata service", "169.254.169.254", false},
		{"a private network", "10.1.2.3", false},
		{"another private network", "192.168.1.10", false},
		{"the third private network", "172.16.4.5", false},
		{"carrier-grade NAT", "100.64.1.1", false},
		{"a v6 link-local", "fe80::1", false},
		{"a v6 unique local", "fd00::1", false},
		{"multicast", "224.0.0.1", false},
		{"not an address at all", "", false},
	} {
		t.Run(c.what, func(t *testing.T) {
			require.Equal(t, c.open, onTheOpenInternet(net.ParseIP(c.ip)),
				"%s (%s) was judged wrongly", c.what, c.ip)
		})
	}
}

// A redirect is a second URL nobody checked.
func TestAPictureIsNotFollowedOffHttps(t *testing.T) {
	client := onlyTheOpenInternet()
	req, err := http.NewRequest(http.MethodGet, "http://authentik.example/face.png", nil)
	require.NoError(t, err)

	err = client.CheckRedirect(req, nil)
	require.Error(t, err, "a redirect to plain http was allowed")
	require.Contains(t, err.Error(), "https")

	safe, err := http.NewRequest(http.MethodGet, "https://authentik.example/face.png", nil)
	require.NoError(t, err)
	require.NoError(t, client.CheckRedirect(safe, nil))

	deep := make([]*http.Request, 5)
	require.Error(t, client.CheckRedirect(safe, deep), "a redirect chain with no end was allowed")
}

// One face, one markup. The rail drew a bare <img class="youface">, which the
// rule that rounds a picture could not reach, so the same face was a circle in
// the conversation and a square in the rooms.
// The same wrapper in every place a face is drawn: the settings page, your own
// turns, and the chip in the bar. A bare <img> is what .youface img cannot
// round, and each of the three has been that at least once.
func TestYourFaceIsTheSameShapeEverywhere(t *testing.T) {
	f := &fakeStore{whoName: "Ronald Lokers", whoFace: []byte("not really a png")}
	f.turns = []squirrel.Turn{{ID: 1, Who: squirrel.SpeakerYou, Words: "the tasks"}}
	m := mounted(t, f)

	said := m.call(t, "GET", "/r/everything", nil).Body.String()
	require.NotContains(t, said, `<img class="youface"`,
		"a turn draws a bare image, which .youface img cannot round")
	require.Equal(t, 1, strings.Count(said, `<span class="youface"`))

	page := m.call(t, "GET", "/me", nil).Body.String()
	require.Equal(t, 1, strings.Count(page, `<span class="youface"`),
		"the settings page draws your face some other way")
}
