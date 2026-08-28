//go:build dev

// devscreen is Squirrel's screen on a port, with invented contents.
//
// It is not a second way to run the product: there is no database, no model,
// and nothing survives the process. What it is for is looking at the screen,
// pressing things, and cutting the network to see what the service worker
// does.
//
//	go run -tags=dev ./cmd/devscreen
//
// See internal/web/dev.go for why the tag is the safety argument.
package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ronaldlokers/squirrel/internal/web"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8420", "where to listen")
	root := flag.String("root", ".", "the checkout to serve from")
	flag.Parse()

	webDir := filepath.Join(*root, "internal", "web")
	if _, err := os.Stat(filepath.Join(webDir, "templates", "layout.html")); err != nil {
		slog.Error("that is not a squirrel checkout", "looked_in", webDir, "error", err)
		os.Exit(1)
	}
	if err := web.DevServe(*addr, webDir, store{}); err != nil {
		slog.Error("the dev screen stopped", "error", err)
		os.Exit(1)
	}
}
