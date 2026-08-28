package web

import (
	"encoding/json"
	"net/http"
	"strings"
)

// The screen is a page you open on a phone, so it may as well be a thing on
// the home screen. Installing changes nothing about what it is: capture stays
// in Campfire, and this remains read-and-triage.
func manifestHandler() http.HandlerFunc {
	// Built rather than embedded, because the icon URLs carry the asset stamp
	// and a static file would have to guess it.
	body, err := json.MarshalIndent(map[string]any{
		"name":        "Squirrel",
		"short_name":  "Squirrel",
		"description": "The pile: read, search and triage what you told Squirrel.",
		// The installed app opens at home, and its scope is the whole screen.
		"start_url":        "/",
		"scope":            "/",
		"display":          "standalone",
		"background_color": "#58388a",
		"theme_color":      "#3b2560",
		"icons": []map[string]string{
			{"src": "/static/icon-192.png?v=" + stamp(), "sizes": "192x192", "type": "image/png"},
			{"src": "/static/icon-512.png?v=" + stamp(), "sizes": "512x512", "type": "image/png", "purpose": "any"},
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

// swHandler serves the worker from the root rather than from /static/, because
// a worker's default scope is the directory it was served from: at
// /static/sw.js it could only ever answer for the assets, which is the one
// thing it does not need to intercept.
func swHandler() http.HandlerFunc {
	source, err := staticFS.ReadFile("static/sw.js")
	if err != nil {
		panic(err)
	}
	// The worker's cache name has to change when the assets do, or a released
	// stylesheet would sit behind a cache keyed to the old one forever.
	body := strings.ReplaceAll(string(source), "SQUIRREL_ASSET_VERSION", stamp())

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		// No Service-Worker-Allowed. This worker comes from /sw.js, so its
		// default scope is already / — every screen, including the one an
		// installed app opens. The header existed to widen a scope by one
		// character when the screen was mounted under a path, and the screen is
		// not mounted under a path any more.
		//
		// Deliberately not the year-long cache the assets get: a worker nobody
		// can replace is a worker you live with forever, and browsers already
		// treat this file as the one they re-check.
		w.Header().Set("Cache-Control", "no-cache")
		_, _ = w.Write([]byte(body))
	}
}
