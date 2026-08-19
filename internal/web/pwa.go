package web

import (
	"encoding/json"
	"net/http"
	"strings"
)

// The screen is a page you open on a phone, so it may as well be a thing on
// the home screen. Installing changes nothing about what it is: capture stays
// in Campfire, and this remains read-and-triage.
func manifestHandler(opts Options) http.HandlerFunc {
	// Built rather than embedded, because start_url and scope are the
	// configured path and a static file would have to guess it.
	body, err := json.MarshalIndent(map[string]any{
		"name":             "Squirrel",
		"short_name":       "Squirrel",
		"description":      "The pile: read, search and triage what you told Squirrel.",
		"start_url":        opts.Path,
		"scope":            opts.Path,
		"display":          "standalone",
		"background_color": "#673ab9",
		"theme_color":      "#3b2560",
		"icons": []map[string]string{
			{"src": opts.Path + "/static/icon-192.png?v=" + assetVersion, "sizes": "192x192", "type": "image/png", "purpose": "any"},
			{"src": opts.Path + "/static/icon-512.png?v=" + assetVersion, "sizes": "512x512", "type": "image/png", "purpose": "any"},
			// A maskable icon is cropped to whatever shape the launcher likes —
			// a circle on most Android homescreens — and only the middle 80% is
			// guaranteed to survive it. The mark reaches nearly the full width
			// of the square it was drawn in, so its tail would be shaved off.
			// This one sits the same art inside its own safe zone, on the same
			// purple, so the padding is invisible.
			{"src": opts.Path + "/static/icon-maskable-512.png?v=" + assetVersion, "sizes": "512x512", "type": "image/png", "purpose": "maskable"},
		},
	}, "", "  ")
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/manifest+json")
		// Short, not a year: it names asset URLs, and it is tiny.
		w.Header().Set("Cache-Control", "public, max-age=300")
		_, _ = w.Write(body)
	}
}

// swHandler serves the worker from the screen's own path rather than from
// /static/, because a worker's default scope is the directory it was served
// from: at /pile/static/sw.js it could only ever answer for the assets, which
// is the one thing it does not need to intercept.
func swHandler(opts Options) http.HandlerFunc {
	source, err := staticFS.ReadFile("static/sw.js")
	if err != nil {
		panic(err)
	}
	// The worker's cache name has to change when the assets do, or a released
	// stylesheet would sit behind a cache keyed to the old one forever.
	body := strings.ReplaceAll(string(source), "SQUIRREL_ASSET_VERSION", assetVersion)

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		// A worker's scope defaults to the directory it came from, and this one
		// came from {path}/sw.js — so it would claim {path}/ and not {path}
		// itself, which is the screen. This header is what lets it widen by one
		// character, and without it the worker installs, reports a healthy
		// scope, and never controls the page anyone actually opens.
		w.Header().Set("Service-Worker-Allowed", opts.Path)
		// Deliberately not the year-long cache the assets get: a worker nobody
		// can replace is a worker you live with forever, and browsers already
		// treat this file as the one they re-check.
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(body))
	}
}
