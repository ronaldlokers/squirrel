package web

import "net/http"

// Stubs. Task 7 replaces both.
func actHandler(s Store, opts Options) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {}
}

func choreHandler(s Store, opts Options) http.HandlerFunc {
	return func(http.ResponseWriter, *http.Request) {}
}
