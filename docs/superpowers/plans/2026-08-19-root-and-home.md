# The root move and the home screen — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Serve the screen from `/` with a two-door home page, move the chores
screen from `/pile/chores` to `/chores`, and delete the configurable mount path.

**Architecture:** `web.Mount` stops composing routes from `opts.Path` and
registers the absolute table the spec pins. Every `{{.Path}}` in every template
becomes a literal absolute URL. A new `homeHandler` renders a template that
takes no data from the store at all — that is the mechanism, not a policy, by
which home renders identically whether the pile is full or empty. `WEB_PATH`,
`Config.WebPath` and `Options.Path` are deleted. The homelab ingress changes in
the same release.

**Tech Stack:** Go 1.26, `html/template`, `embed`, `net/http.ServeMux` (Go 1.22
method+pattern routing), testify, headless Chromium over CDP for the browser
tests (build tag `browser`).

**Spec:** `docs/superpowers/specs/2026-08-19-root-and-home-design.md`

**Comps (normative):** `.impeccable/comps/home-screen.html` and its notes file
`.impeccable/comps/home-screen.md`.

## Global Constraints

- **Never a count, in any form.** Not a number, not "3 of 12", not a dot, not a
  badge, not a stack of cards behind another card. This is the product's
  hardest rule and it applies to the home screen exactly as it applies to the
  deck.
- **Never a capture box.** No route creates an item. The home screen's foot
  line — `thoughts go in through the chat` — is the whole of what it says about
  capture.
- **Two views must agree.** Nothing on the home screen can be acted on, so the
  screen and the chat can never disagree about a note.
- **The route table in the spec is normative.** Copy it exactly; do not invent
  or rename a route.
- **Identity:** everything that can read a note stays behind `guard`. The
  assets, the manifest and the worker answer without an identity. That
  arrangement took three releases to get right (v0.9.1–v0.9.3) — re-path it,
  never re-litigate it.
- **No JavaScript is required by the home screen.** Plain links and one bare
  form.
- **No new design tokens.** Colours, radii, type steps and shadow values all
  come from `DESIGN.md`; the comp introduces no new ones.
- Commit subjects are conventional-commit style, lowercase, imperative:
  `feat: ...`, `fix: ...`, `docs: ...`.

## File structure

| File | Responsibility after this plan |
| --- | --- |
| `internal/web/web.go` | `Options` without `Path`; the `Store` interface. |
| `internal/web/pile.go` | `Mount` with the absolute route table; the deck handler. |
| `internal/web/home.go` | **new** — `homeHandler`, which reads nothing. |
| `internal/web/templates/home.html` | **new** — the two doors and the foot line. |
| `internal/web/templates/layout.html` | The lid, now with absolute URLs, a brand that links home, and no cross-link on home. |
| `internal/web/render.go` | `view` gains `Home bool`; `pages` gains `"home"`. |
| `internal/web/pwa.go` | `start_url`/`scope` become `/`; `Service-Worker-Allowed` goes. |
| `internal/web/assets.go` | Static prefix becomes the literal `/static/`. |
| `internal/web/static/door-pile.png`, `door-chores.png` | **new** — the door art, sized for the slot. |
| `internal/web/static/pile.css` | Gains the home screen's rules from the comp. |
| `internal/squirrel/config.go` | `WebPath` deleted. |
| `internal/boot/boot.go` | Stops passing `Path`. |
| `docs/pile-screen.md` | Documents the new table; drops `WEB_PATH`. |
| homelab `apps/{production,staging}/squirrel/*` | Ingress paths re-pathed. |

---

### Task 1: The screen answers at absolute paths

Everything except the home page itself. After this task `/pile` is still the
deck, but `/chores` is the chores screen, `/static/` serves the assets, and
nothing is composed from a configured prefix.

**Files:**
- Modify: `internal/web/web.go` (delete `Options.Path`)
- Modify: `internal/web/pile.go` (`Mount`, `pileHandler`, `searchInto`)
- Modify: `internal/web/chores.go` (the `view{Path: opts.Path…}` literals)
- Modify: `internal/web/act.go` (redirect targets)
- Modify: `internal/web/assets.go` (`staticHandler` prefix)
- Modify: `internal/web/pwa.go` (`start_url`, `scope`, icon URLs, drop the header)
- Modify: `internal/web/render.go` (delete `view.Path`)
- Modify: `internal/web/templates/{layout,card,bottom,empty,every,pile,results,chores}.html`
- Modify: `internal/boot/boot.go:199` (stop passing `Path`)
- Test: `internal/web/pile_test.go`, `internal/web/testsupport_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks.
- Produces: `web.Options{IdentityHeader, Identity string; Owner func() int64}` —
  no `Path` field. `Mount(m Mux, s Store, opts Options) error` unchanged in
  signature. The `view` struct loses its `Path` field.

- [ ] **Step 1: Write the failing route-table test**

Add to `internal/web/pile_test.go`:

```go
// TestTheRouteTable pins the shape of the screen rather than describing it.
// The table is the spec's table; a route that moves has to move here first.
func TestTheRouteTable(t *testing.T) {
	m := mounted(t, &fakeStore{items: []squirrel.Item{note(1, "a thought", squirrel.ItemOpen)}})

	for _, route := range []string{
		"GET /{$}",
		"GET /pile",
		"POST /pile/act",
		"POST /pile/chore",
		"GET /chores",
		"POST /chores/act",
		"GET /pile/chores",
		"GET /manifest.webmanifest",
		"GET /sw.js",
		"GET /static/",
	} {
		require.Contains(t, m.routes, route, "the route table lost %s", route)
	}
	require.Len(t, m.routes, 10, "a route was added without being pinned here")
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/web/ -run TestTheRouteTable -v`
Expected: FAIL — `the route table lost GET /{$}` (and the old `/pile/...`
patterns are still registered).

- [ ] **Step 3: Teach the test mux about `/{$}`**

`testsupport_test.go`'s `call` does prefix matching, which would let `/`
answer for everything. Go's real `ServeMux` treats `/{$}` as an exact match;
the helper has to do the same or the test is a lie. In
`internal/web/testsupport_test.go`, replace the matching loop inside `call`:

```go
	best := ""
	for pattern := range m.routes {
		wantMethod, path, _ := strings.Cut(pattern, " ")
		if wantMethod != method {
			continue
		}
		// `{$}` is Go's "this path and nothing under it". Prefix matching
		// would let / answer for every URL on the screen.
		if exact, ok := strings.CutSuffix(path, "{$}"); ok {
			if target != exact {
				continue
			}
		} else if !strings.HasPrefix(target, path) {
			continue
		}
		if len(pattern) > len(best) {
			best = pattern
		}
	}
```

and update `mounted` to stop setting `Path`:

```go
func mounted(t *testing.T, f *fakeStore) *testMux {
	t.Helper()
	m := newTestMux()
	require.NoError(t, Mount(m, f, Options{
		IdentityHeader: "X-Authentik-Username", Identity: "ronald",
		Owner: func() int64 { return 1 },
	}))
	return m
}
```

- [ ] **Step 4: Delete `Options.Path`**

In `internal/web/web.go`, remove the `Path` field and its comment. `Options`
becomes `IdentityHeader`, `Identity`, `Owner`.

- [ ] **Step 5: Rewrite `Mount`'s table**

In `internal/web/pile.go`:

```go
func Mount(m Mux, s Store, opts Options) error {
	if opts.Identity == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_IDENTITY is empty")
	}
	if opts.Owner == nil {
		return fmt.Errorf("refusing to mount the pile: no owner")
	}
	// `{$}` and not `/`: a bare "/" pattern is Go's catch-all, and the home
	// screen would answer for every URL nobody else claimed — including the
	// typos, which would then look like a working page.
	m.Get("/{$}", guard(opts, homeHandler()))
	m.Get("/pile", guard(opts, pileHandler(s, opts)))
	// Both writes carry the origin check as well as the identity one: the
	// identity says who is asking, sameOrigin says which page asked.
	m.Post("/pile/act", guard(opts, sameOrigin(actHandler(s, opts))))
	m.Post("/pile/chore", guard(opts, sameOrigin(choreHandler(s, opts))))
	m.Get("/chores", guard(opts, choresHandler(s, opts)))
	m.Post("/chores/act", guard(opts, sameOrigin(choreActHandler(s, opts))))
	// The chores screen lived here for its whole life. A bookmark that dies
	// quietly is worse than a redirect nobody notices.
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/chores", http.StatusMovedPermanently)
	}))
	// Outside the guard, like the worker below and for the same reason: a
	// browser fetches a manifest without the cookies that carry the identity,
	// and one that answers 403 leaves an installed app with no icon and no
	// explanation. It names the app and lists four PNGs — there is nothing in
	// it to protect.
	m.Get("/manifest.webmanifest", manifestHandler())
	// Not behind the guard: a browser fetches the worker without the cookies
	// that carry the identity, and a worker that 302s to a login page is a
	// worker that never installs. It contains no notes — only which files to
	// keep and what to say when the network is gone.
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}
```

`homeHandler` does not exist until Task 4. For this task only, add a
placeholder in `internal/web/home.go` that Task 4 replaces:

```go
package web

import "net/http"

func homeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, "home", view{Home: true})
	}
}
```

and, in `render.go`, add `Home bool` to `view` plus a `"home"` entry to
`pages` pointing at a template Task 4 writes. To keep this task's tree
compiling and green, create `internal/web/templates/home.html` now with the
minimum the layout needs, and let Task 4 replace its body:

```html
{{define "content"}}<div class="homebox"></div>{{end}}
```

- [ ] **Step 6: Drop the path from every handler and template**

`pileHandler`, `searchInto`, `choresHandler`, `actHandler`, `choreHandler`,
`choreActHandler`: delete `Path: opts.Path` from every `view{...}` literal and
replace every redirect target built from `opts.Path` with the literal it now
is (`/pile`, `/chores`). In the templates, rewrite every URL:

| Was | Becomes |
| --- | --- |
| `{{.Path}}/static/…` | `/static/…` |
| `{{.Path}}/manifest.webmanifest` | `/manifest.webmanifest` |
| `{{.Path}}` (the deck) | `/pile` |
| `{{.Path}}/chores` | `/chores` |
| `{{.Path}}/act`, `{{$.Path}}/act` | `/pile/act` |
| `{{.Path}}/chore` | `/pile/chore` |
| `{{.Path}}/chores/act` | `/chores/act` |

Grep afterwards: `grep -rn '{{\.Path}}\|{{\$\.Path}}' internal/web/templates/`
must print nothing.

- [ ] **Step 7: Re-path the assets, the manifest and the worker**

`internal/web/assets.go` — `staticHandler` takes no options:

```go
func staticHandler() http.HandlerFunc {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	files := http.FileServer(http.FS(sub))
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000")
		http.StripPrefix("/static/", files).ServeHTTP(w, r)
	}
}
```

`internal/web/pwa.go` — `manifestHandler()` and `swHandler()` take no options.
`start_url` and `scope` become `"/"`; the icon `src` values become
`"/static/icon-192.png?v=" + assetVersion` and the 512 equivalent. Delete the
`Service-Worker-Allowed` header and replace its comment with why it is gone:

```go
		// No Service-Worker-Allowed: this worker is served from /sw.js, so its
		// default scope is already /. The header existed to widen a scope by
		// one character when the screen was mounted under a path, and the
		// screen is not mounted under a path any more.
```

- [ ] **Step 8: Stop passing the path from boot**

`internal/boot/boot.go` around line 199:

```go
		if err := web.Mount(server, store, web.Options{
			IdentityHeader: config.WebIdentityHeader,
			Identity:       config.WebIdentity,
			Owner:          webOwner.Load,
		}); err != nil {
			cancel()
			return nil, fmt.Errorf("mounting the pile: %w", err)
		}
		slog.Info("the screen is mounted", "at", "/")
```

- [ ] **Step 9: Run the package's tests and fix the fallout**

Run: `go build ./... && go test ./internal/web/ ./internal/boot/ ./internal/squirrel/`

Expected failures to fix, all mechanical: `internal/boot/testsupport_test.go:266`
(`pileURL` — point it at `/pile`), any test asserting `/pile/static/` or
`/pile/chores` in a body or a `Location` header, and `pwa_test.go`'s
assertions about `start_url`, `scope` and the worker header. Where a test
asserted the `Service-Worker-Allowed` header, replace it rather than delete
it:

```go
func TestTheWorkerNeedsNoWidenedScope(t *testing.T) {
	m := mounted(t, &fakeStore{})
	w := m.call(t, "GET", "/sw.js", nil)

	require.Equal(t, 200, w.Code)
	require.Empty(t, w.Header().Get("Service-Worker-Allowed"),
		"a worker served from /sw.js already scopes to /")
}
```

- [ ] **Step 10: Test the redirect**

Add to `internal/web/pile_test.go`:

```go
func TestTheOldChoresURLRedirects(t *testing.T) {
	m := mounted(t, &fakeStore{})
	w := m.call(t, "GET", "/pile/chores", nil)

	require.Equal(t, http.StatusMovedPermanently, w.Code)
	require.Equal(t, "/chores", w.Header().Get("Location"))
}
```

Run: `go test ./internal/web/ -run TestTheOldChoresURLRedirects -v`
Expected: PASS.

- [ ] **Step 11: Full suite**

Run: `go test ./...`
Expected: PASS.

- [ ] **Step 12: Commit**

```bash
git add internal/ docs/
git commit -m "feat: serve the screen from the root"
```

---

### Task 2: `WEB_PATH` leaves the configuration

**Files:**
- Modify: `internal/squirrel/config.go:76-77,254`
- Modify: `internal/squirrel/config_test.go:130,139-144`
- Modify: `docs/pile-screen.md:18,161-169`

**Interfaces:**
- Consumes: Task 1's `Options` without `Path`.
- Produces: `squirrel.Config` without a `WebPath` field.

- [ ] **Step 1: Delete the assertions that pin the setting**

In `internal/squirrel/config_test.go`, delete the `require.Equal(t, "/pile",
cfg.WebPath)` line from the defaults test and the `"WEB_PATH": "/notes"` entry
plus its `require.Equal(t, "/notes", cfg.WebPath)` from the override test.

- [ ] **Step 2: Run and watch it fail to compile**

Run: `go test ./internal/squirrel/ -run TestConfig`
Expected: PASS (the field still exists; nothing reads it). This step is the
check that nothing else asserted on it.

- [ ] **Step 3: Delete the field and its read**

`internal/squirrel/config.go`: remove the `WebPath string` field with its
comment, and remove the `WebPath: optional(env, "WEB_PATH", "/pile"),` line
from the constructor.

- [ ] **Step 4: Build**

Run: `go build ./... && go test ./...`
Expected: PASS. A compile error here names a caller Task 1 missed.

- [ ] **Step 5: Update the docs**

In `docs/pile-screen.md`, delete the `WEB_PATH` row from the environment
table, and rewrite the worker section that explains `Service-Worker-Allowed`.
Replace the `{WEB_PATH}` prose with:

```markdown
**The worker is served from `/sw.js`, not from `/static/sw.js`.** A worker's
scope defaults to the directory it was served from, so one served out of
`/static/` could only ever answer for the assets — the one thing it does not
need to intercept. From `/sw.js` it scopes to `/` and controls every screen,
which needs no extra header to permit it.
```

Add the spec's route table to the same document, verbatim.

- [ ] **Step 6: Commit**

```bash
git add internal/squirrel/ docs/
git commit -m "feat: drop the configurable mount path"
```

---

### Task 3: The door art ships as a static asset

The two illustrations live in `.impeccable/comps/assets/` at 1.2 MB, 170 kB
and 158 kB. The comp may carry originals; the binary may not.

**Files:**
- Create: `internal/web/static/door-pile.png`
- Create: `internal/web/static/door-chores.png`
- Test: `internal/web/assets_test.go`

**Interfaces:**
- Consumes: Task 1's `staticHandler()`.
- Produces: `/static/door-pile.png` and `/static/door-chores.png`, served with
  the year-long cache and the `?v=` stamp like every other asset.

- [ ] **Step 1: Write the failing test**

Add to `internal/web/assets_test.go`:

```go
// The door art is part of the screen, not part of the comp: a home page whose
// illustrations 404 is a home page with two empty slots.
func TestTheDoorArtIsServed(t *testing.T) {
	m := mounted(t, &fakeStore{})

	for _, name := range []string{"door-pile.png", "door-chores.png"} {
		w := m.call(t, "GET", "/static/"+name, nil)
		require.Equal(t, 200, w.Code, name)
		require.NotEmpty(t, w.Body.Bytes(), name)
	}
}
```

- [ ] **Step 2: Run it and watch it fail**

Run: `go test ./internal/web/ -run TestTheDoorArtIsServed -v`
Expected: FAIL — 404.

- [ ] **Step 3: Resize the two drawings into the static directory**

The art slot is 86 px tall on a desktop and 66 px on a phone, so 256 px tall
covers a 3× screen with room to spare. Keep the aspect ratio; keep the
transparency.

```bash
magick .impeccable/comps/assets/door-pile.png \
  -resize x256 -strip internal/web/static/door-pile.png
magick .impeccable/comps/assets/door-chores.png \
  -resize x256 -strip internal/web/static/door-chores.png
ls -l internal/web/static/door-*.png
```

If `magick` is absent, `mise use -g imagemagick` or use
`convert` from ImageMagick 6. Expected: both files well under 60 kB.

- [ ] **Step 4: Run the test**

Run: `go test ./internal/web/ -run TestTheDoorArtIsServed -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/web/static/door-pile.png internal/web/static/door-chores.png internal/web/assets_test.go
git commit -m "feat: ship the door art with the binary"
```

---

### Task 4: The home screen

**Files:**
- Modify: `internal/web/home.go` (replace Task 1's placeholder)
- Modify: `internal/web/templates/home.html` (replace Task 1's placeholder)
- Modify: `internal/web/templates/layout.html` (brand links home; no cross-link on home)
- Modify: `internal/web/static/pile.css` (the comp's home rules)
- Test: `internal/web/home_test.go` (new)

**Interfaces:**
- Consumes: `view{Home bool}` and the `"home"` page from Task 1;
  `/static/door-*.png` from Task 3.
- Produces: `homeHandler() http.HandlerFunc` — note the absent `Store`
  parameter, which is the point.

- [ ] **Step 1: Write the failing tests**

Create `internal/web/home_test.go`:

```go
package web

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// The load-bearing test. Home reads nothing, so a full pile and an empty one
// are the same bytes — which is the guard against this screen growing a
// preview, a badge or a count by accident. Any of those would break this.
func TestHomeIsTheSameWhateverThePileHolds(t *testing.T) {
	full := mounted(t, &fakeStore{items: []squirrel.Item{
		note(1, "the tyre is flat", squirrel.ItemOpen),
		note(2, "ring the dentist", squirrel.ItemOpen),
	}})
	empty := mounted(t, &fakeStore{})

	require.Equal(t,
		empty.call(t, "GET", "/", nil).Body.String(),
		full.call(t, "GET", "/", nil).Body.String())
}

// Home fails to fail: there is nothing on it that needs the database, so an
// unreachable one is not this screen's problem to report.
func TestHomeStandsUpWithoutTheDatabase(t *testing.T) {
	m := mounted(t, &fakeStore{err: errTest})
	w := m.call(t, "GET", "/", nil)

	require.Equal(t, 200, w.Code)
	require.Contains(t, w.Body.String(), "the pile")
	require.Contains(t, w.Body.String(), "the chores")
}

func TestHomeHasTwoDoorsAndNothingElseToPress(t *testing.T) {
	m := mounted(t, &fakeStore{})
	body := m.call(t, "GET", "/", nil).Body.String()

	require.Equal(t, 1, strings.Count(body, `href="/pile"`), "one door to the pile")
	require.Equal(t, 1, strings.Count(body, `href="/chores"`), "one door to the chores")
	// The lid's cross-link is the third copy of a door. Not on this screen.
	require.NotContains(t, body, `class="lidlink"`)
	require.Contains(t, body, "thoughts go in through the chat")
}

// Everywhere else, the mark is the way back. Home is where it points, so on
// home it is not a link.
func TestTheMarkGoesHomeFromTheOtherScreens(t *testing.T) {
	m := mounted(t, &fakeStore{})

	require.Contains(t, m.call(t, "GET", "/pile", nil).Body.String(), `<a class="brand" href="/"`)
	require.Contains(t, m.call(t, "GET", "/chores", nil).Body.String(), `<a class="brand" href="/"`)
	require.NotContains(t, m.call(t, "GET", "/", nil).Body.String(), `<a class="brand"`)
}

// The screen-wide rules do not stop at the deck.
func TestHomeNeverEmitsACount(t *testing.T) {
	m := mounted(t, &fakeStore{items: []squirrel.Item{
		note(1, "one", squirrel.ItemOpen),
		note(2, "two", squirrel.ItemOpen),
		note(3, "three", squirrel.ItemOpen),
	}})
	body := strings.ToLower(m.call(t, "GET", "/", nil).Body.String())

	for _, count := range []string{"3 notes", "3 more", "(3)", "1 of ", "waiting"} {
		require.NotContains(t, body, count)
	}
	require.NotContains(t, body, "<textarea")
	require.NotContains(t, body, `type="text"`)
}
```

- [ ] **Step 2: Run them and watch them fail**

Run: `go test ./internal/web/ -run TestHome -v`
Expected: FAIL — the placeholder template has no doors.

- [ ] **Step 3: Write the home template**

Replace `internal/web/templates/home.html` with the comp's markup. The comp is
normative; this is it with the URLs made real:

```html
{{define "content"}}
<div class="homebox">
  {{/* The two halves of what the box holds, and they are equals: one grid,
       two identical cells, the same card stock, the same depth. Neither leads,
       at any width — a stacked pair would read as first and second. */}}
  <nav class="doors" aria-label="The two halves of the box">
    <a class="door" href="/pile">
      {{/* Decorative: the door's name is directly below it. */}}
      <span class="art"><img alt="" src="/static/door-pile.png?v={{.V}}" width="342" height="256"></span>
      <span class="label">
        <span class="name">the pile</span>
        <span class="what">what you said</span>
      </span>
    </a>
    <a class="door" href="/chores">
      <span class="art"><img alt="" src="/static/door-chores.png?v={{.V}}" width="326" height="256"></span>
      <span class="label">
        <span class="name">the chores</span>
        <span class="what">what comes back</span>
      </span>
    </a>
  </nav>

  {{/* The first screen of an installed app is where a capture box would try
       to grow. This states where capture lives instead, and asks nothing. */}}
  <p class="hint">thoughts go in through the chat</p>
</div>
{{end}}
```

Set the two `width`/`height` attributes to the actual pixel dimensions the
files got in Task 3 (`identify internal/web/static/door-pile.png`), so the
slot does not reflow while the image loads.

- [ ] **Step 4: Make the lid know where it is**

In `internal/web/templates/layout.html`, replace the brand block:

```html
  {{if .Home}}
  <span class="brand">
    <img alt="" src="/static/logo.png?v={{.V}}" width="84" height="63">
    <span class="wordmark">Squirrel</span>
  </span>
  {{else}}
  {{/* The convention every website has, and the cheapest possible way back. */}}
  <a class="brand" href="/">
    <img alt="" src="/static/logo.png?v={{.V}}" width="84" height="63">
    <span class="wordmark">Squirrel</span>
  </a>
  {{end}}
```

and wrap the cross-link so home has none:

```html
  {{/* The Lid Link offers the half you are not looking at, so moving between
       the pile and the chores never goes via home. On home itself there is no
       cross-link: both doors are already on the page, and a third copy of a
       door is furniture. */}}
  {{if not .Home}}
    {{if eq .Here "chores"}}
    <a class="lidlink" href="/pile">the pile</a>
    {{else}}
    <a class="lidlink" href="/chores">chores</a>
    {{end}}
  {{end}}
```

The search form's action becomes `/pile` — results already live there.

- [ ] **Step 5: Give the handler its comment**

`internal/web/home.go`:

```go
package web

import "net/http"

// The home screen takes no Store, and that absence is the design.
//
// A home screen that shows what is waiting greets you with what is waiting,
// however carefully it is dressed — so this one shows nothing that depends on
// what the pile holds. A full pile and an empty one render the same bytes,
// which also means there is no state here to disagree with the chat about, and
// nothing on the page to triage.
//
// It answers even when Postgres does not, for the same reason.
func homeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		render(w, "home", view{Home: true})
	}
}
```

- [ ] **Step 6: Port the comp's CSS**

Append to `internal/web/static/pile.css`, taken from
`.impeccable/comps/home-screen.html` verbatim — the `.homebox`, `.doors`,
`.door`, `.door .art`, `.door .label`, `.door .name`, `.door .what` and
`.hint` rules, plus their `@media (max-width: 620px)` counterparts. Do **not**
port the `.peek*` rules: the peek was cut. Do not port `.viewport` or
`.compnote`: comp chrome, not product.

The home screen centres its content, which the deck does not, so scope the
centring rather than changing `main` globally:

```css
/* Home is the one screen with nothing above or below its content, so it sits
   in the middle of the field rather than at the top of it. */
main:has(.homebox) {
  display: flex; justify-content: center; align-items: center;
  padding: 58px 22px 48px;
}
```

The brand is a link on every screen but home, and a link must not look like
one here — the mark and wordmark are the mark and wordmark:

```css
.brand { text-decoration: none; }
```

- [ ] **Step 7: Run the tests**

Run: `go test ./internal/web/ -run TestHome -v && go test ./internal/web/ -v`
Expected: PASS.

- [ ] **Step 8: Look at it**

Run the binary against the local Postgres and open both widths:

```bash
go run ./cmd/squirrel &
```

Check by eye, in one pass, at 1280 px and at 390 px: the two doors are equal
in size and weight, the art is the same height on both, neither drawing is
clipped, the foot line sits under them, the lid carries no cross-link, and the
mark is not underlined. Fix everything that pass shows in one batch. Do not
iterate further.

- [ ] **Step 9: Commit**

```bash
git add internal/web/
git commit -m "feat: open on a screen with two doors"
```

---

### Task 5: The worker controls the root

**Files:**
- Modify: `internal/web/browser_test.go`

**Interfaces:**
- Consumes: everything above.
- Produces: nothing.

- [ ] **Step 1: Move the existing worker test to the home page**

`TestBrowserTheWorkerTakesTheScreen` currently navigates to `/pile` twice and
asserts the second load was served from the worker's cache. Point it at `/`
and assert the scope:

```go
	// The worker is served from /sw.js, so it scopes to / and controls every
	// screen — including this one, which is the URL an installed app opens.
	scope := eval(t, page, `navigator.serviceWorker.ready.then(r => r.scope)`)
	require.True(t, strings.HasSuffix(scope, "/"), "scope was %q", scope)
	require.False(t, strings.Contains(scope, "/pile"), "scope was %q", scope)
```

- [ ] **Step 2: Move the other browser tests with their screens**

Any browser test navigating to `/pile/chores` becomes `/chores`. Grep:
`grep -n '/pile' internal/web/browser_test.go internal/web/cdp_test.go`.
Every remaining `/pile` should be the deck itself.

- [ ] **Step 3: Run the browser suite**

Run: `go test -tags browser ./internal/web/ -v`
Expected: PASS. (Chromium needs `--disable-dev-shm-usage`; the harness
already passes it.)

- [ ] **Step 4: Commit**

```bash
git add internal/web/browser_test.go
git commit -m "test: the worker takes the root"
```

---

### Task 6: Release

**Files:** none in this repo beyond the tag.

- [ ] **Step 1: Full suite, both tags**

Run: `go test ./... && go test -tags browser ./internal/web/`
Expected: PASS.

- [ ] **Step 2: Open the pull request**

```bash
git push -u origin feat/the-root-and-the-front-door
gh pr create --title "feat: open on a home screen at the root" --body "..."
```

The body must say, in its own words: the ingress changes with it, so the
homelab PR merges only after the image exists, and neither may land alone.

- [ ] **Step 3: Merge once CI is green, then tag**

```bash
gh pr checks --watch
gh pr merge --squash --delete-branch
git checkout main && git pull
git tag -a v0.10.0 -m "feat: open on a home screen at the root"
git push origin v0.10.0
```

`v0.10.0` and not `v0.9.5`: the URL of every screen changed.

- [ ] **Step 4: Wait for the image**

Run: `gh run watch`
Expected: the `image` job succeeds. The homelab change is not opened before
this passes — an ingress pointing at `/` in front of a binary that serves
`/pile` is a 404 on the only page there is.

---

### Task 7: The edge follows

**Files (repo: `~/Projects/github.com/ronaldlokers/homelab`):**
- Modify: `apps/production/squirrel/ingress-pile.yaml`
- Modify: `apps/production/squirrel/ingress-assets.yaml`
- Modify: `apps/production/squirrel/kustomization.yaml`
- Modify: `apps/staging/squirrel/ingress.yaml`
- Modify: `apps/staging/squirrel/kustomization.yaml`

- [ ] **Step 1: Branch**

```bash
cd ~/Projects/github.com/ronaldlokers/homelab
git checkout main && git pull
git checkout -b feat/squirrel-at-the-root
```

- [ ] **Step 2: Widen the screen's route**

In `apps/production/squirrel/ingress-pile.yaml`, the path becomes `/` and the
resource is renamed for what it now is. Keep both middlewares. Rewrite the
head comment to say what the rule now means:

```yaml
# The screen, all of it: home at /, the deck at /pile, the chores at /chores.
#
# One broad rule rather than an enumeration — every new route would otherwise
# need a change here, and forgetting one produces a 404 that looks like an
# application bug. Traefik routes on the longest matching prefix, so the
# narrower rules (the assets, the outpost, the presence hook) keep winning
# without this one having to know about them.
#
# Two paths become reachable from outside that were not before:
# /transports/campfire and /healthz. Both now sit behind Authentik, so an
# anonymous caller gets a login page rather than a webhook. Campfire delivers
# in-cluster and never traverses this ingress; the kubelet's probes are
# in-cluster too.
```

with:

```yaml
          - path: /
            pathType: Prefix
```

- [ ] **Step 3: Re-path the assets**

In `apps/production/squirrel/ingress-assets.yaml`: `/pile/static/` becomes
`/static/`, `/pile/manifest.webmanifest` becomes `/manifest.webmanifest`, and
add the worker, which the spec's table puts here:

```yaml
          - path: /sw.js
            pathType: Exact
            backend:
              service:
                name: squirrel
                port:
                  number: 8080
```

Update the head comment's last paragraph, which names `/pile/static/`.

- [ ] **Step 4: Staging**

`apps/staging/squirrel/ingress.yaml`: `/pile` becomes `/`. Staging's identity
middleware injects a header rather than redirecting, so staging needs no
separate assets rule — and did not have one.

- [ ] **Step 5: Bump both images**

`newTag: v0.10.0` in both `apps/production/squirrel/kustomization.yaml:62`
and `apps/staging/squirrel/kustomization.yaml:44`.

- [ ] **Step 6: Open, merge, reconcile**

```bash
git add apps/
git commit -m "feat: serve squirrel's screen from the root"
git push -u origin feat/squirrel-at-the-root
gh pr create --title "feat: serve squirrel's screen from the root" --body "..."
gh pr merge --squash --delete-branch
flux reconcile kustomization apps-staging --with-source
flux reconcile kustomization apps-production --with-source
```

- [ ] **Step 7: Verify both, from outside**

```bash
kubectl -n campfire get pods -l app=squirrel
curl -sI https://squirrel.staging.ronaldlokers.nl/ | head -1
curl -sI https://squirrel.ronaldlokers.nl/static/door-pile.png | head -1
curl -sI https://squirrel.ronaldlokers.nl/pile/chores | head -2
```

Expected: staging `200`; the art `200`; the old chores URL `301` with
`location: /chores`. Then open `https://squirrel.ronaldlokers.nl/` in a
browser and press both doors.

---

## Self-review

**Spec coverage.** Route table — Task 1 (test) and Task 7 (edge). `WEB_PATH`
removal — Task 2. Home screen, no preview, no count — Task 4. `Service-Worker-Allowed`
removal — Task 1 step 7. Lid keeps its cross-link off home, mark links home —
Task 4 step 4. Database down — Task 4's `TestHomeStandsUpWithoutTheDatabase`.
Identical empty/full — Task 4's first test. Worker scope — Task 5. Rollout
ordering — Task 6 step 4 and Task 7. The `/pile/chores` 301 — Task 1 step 10.
The installed app keeps working because `start_url` `/pile` still serves the
deck — no task needed; it follows from `/pile` being unchanged, and Task 1's
route table test pins it.

**One gap, deliberately left:** the `DESIGN.md` amendments the comp's notes
propose (the Door, the Door Art and its guard rails, the Lid Link line, the
layout note, and the chores-art exception) are their own commit, made before
this plan is executed, because they are the record of a decision rather than a
step in building it.
