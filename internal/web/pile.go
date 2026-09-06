package web

import (
	"fmt"
	"net/http"
)

const pileLimit = 1

type Mux interface {
	Get(pattern string, h http.HandlerFunc)
	Post(pattern string, h http.HandlerFunc)
}

func posting(opts Options, h http.HandlerFunc) http.HandlerFunc {
	return guard(opts, sameOrigin(h))
}

func Mount(m Mux, s Store, opts Options) error {
	if opts.RequiredGroup == "" {
		return fmt.Errorf("refusing to mount the pile: WEB_REQUIRED_GROUP is empty")
	}
	m = knowsYou(m, s, opts.Location)
	if opts.Gate == nil {
		return fmt.Errorf("refusing to mount the pile: no way in")
	}
	if opts.Sessions == nil {
		return fmt.Errorf("refusing to mount the pile: no sessions")
	}
	if opts.Login == nil {
		return fmt.Errorf("refusing to mount the pile: nothing turns a login into a person")
	}
	m.Get("/{$}", guard(opts, boardHandler(s, opts)))
	m.Get("/r/{room}", guard(opts, roomRoute(opts)))
	m.Post("/capture", posting(opts, captureHandler(s, opts)))
	m.Get("/board", guard(opts, boardHandler(s, opts)))
	m.Post("/board/act", posting(opts, boardActHandler(s, opts)))
	m.Post("/board/undo", posting(opts, boardUndoHandler(s, opts)))
	m.Post("/board/new", posting(opts, boardNewHandler(s, opts)))
	m.Post("/board/now", posting(opts, boardNowHandler(s, opts)))
	m.Post("/board/capture", posting(opts, boardCaptureHandler(s, opts)))
	m.Post("/board/chore", posting(opts, boardChoreHandler(s, opts)))
	m.Post("/board/mood", posting(opts, boardMoodHandler(s)))
	m.Post("/board/notuseful", posting(opts, boardNotUsefulHandler(s)))
	m.Post("/board/ask", posting(opts, boardAskHandler(s, opts)))
	m.Post("/board/fix", posting(opts, boardFixHandler(s, opts)))
	m.Get("/me", guard(opts, meHandler(s, opts)))
	m.Get("/me/face", guard(opts, faceHandler(s)))
	m.Post("/me/forget", guard(opts, sameOrigin(meForgetHandler(s))))
	if opts.Photos != nil {
		m.Get("/photo/{id}", guard(opts, photoHandler(s, opts)))
		m.Get("/photo/{id}/thumb", guard(opts, thumbHandler(s, opts)))
		m.Get("/photo/{id}/checksum", guard(opts, photoChecksumHandler(s, opts)))
	}
	if opts.PushKey != "" {
		m.Post("/push/subscribe", guard(opts, sameOrigin(pushSubscribeHandler(s, opts))))
		m.Post("/push/forget", guard(opts, sameOrigin(pushForgetHandler(s, opts))))
	}
	for _, gone := range []string{"/coach", "/buddy"} {
		m.Get(gone, guard(opts, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/", http.StatusMovedPermanently)
		}))
	}
	m.Post("/steps", posting(opts, stepsHandler(s, opts)))
	m.Get("/at/{id}", guard(opts, atOneHandler(s, opts)))
	for _, gone := range []string{"/moods", "/knowing"} {
		m.Get(gone, guard(opts, func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/me", http.StatusMovedPermanently)
		}))
	}
	m.Get("/pile/chores", guard(opts, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/?bay=chores", http.StatusMovedPermanently)
	}))
	m.Post("/timer", posting(opts, timerHandler(s, opts)))
	m.Get("/auth", gateHandler())
	m.Post("/auth/in", sameOrigin(beginHandler(opts)))
	m.Get("/auth/callback", backHandler(opts))
	m.Post("/auth/out", sameOrigin(outHandler(opts)))
	m.Get("/manifest.webmanifest", manifestHandler())
	m.Get("/sw.js", swHandler())
	m.Get("/static/", staticHandler())
	return nil
}

const searchLimit = 6

const choreHits = 3
