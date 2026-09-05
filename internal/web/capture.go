package web

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

const captureLimit = 4000

const keepingTakesAtMost = 8 * time.Second

func captureHandler(s Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		personID, ok := personOf(r)
		if !ok {
			fail(w, errNoOwner)
			return
		}
		slog.Info("a capture arrived",
			"content_type", r.Header.Get("Content-Type"),
			"bytes", r.ContentLength,
			"transport", "screen")

		text, photo, kind, key, err := readCapture(r, opts)
		if err != nil {
			slog.Warn("a capture was refused", "error", err)
			fail(w, err)
			return
		}

		if text == "" && photo == "" {
			slog.Info("a capture had nothing in it",
				"content_type", r.Header.Get("Content-Type"), "bytes", r.ContentLength)
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}

		sender := subOf(r)
		slog.Info("a capture is being kept",
			"photograph", photo != "", "kind", kind, "words", len(text) > 0)

		ctx, done := context.WithTimeout(r.Context(), keepingTakesAtMost)
		defer done()

		if _, err := s.InsertItemReturningID(ctx, squirrel.Item{
			Transport:  squirrel.ScreenTransport,
			SenderID:   &sender,
			PersonID:   &personID,
			RawText:    text,
			Payload:    []byte(squirrel.ScreenCapture),
			ReceivedAt: time.Now(),
			PhotoName:  photo,
			PhotoType:  kind,
			CaptureKey: key,
		}); err != nil {
			slog.Warn("a capture could not be kept", "error", err)
			fail(w, err)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

var errNotAPhotograph = errors.New("not a photograph this keeps")

func readCapture(r *http.Request, opts Options) (text, photo, kind, key string, err error) {
	parts, err := r.MultipartReader()
	if errors.Is(err, http.ErrNotMultipart) {
		if err := r.ParseForm(); err != nil {
			return "", "", "", "", fmt.Errorf("reading what you said: %w", err)
		}
		return said(r.FormValue("text")), "", "", strings.TrimSpace(r.FormValue("key")), nil
	}
	if err != nil {
		return "", "", "", "", fmt.Errorf("reading what you sent: %w", err)
	}

	for {
		part, err := parts.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return text, photo, kind, key, fmt.Errorf("reading what you sent: %w", err)
		}

		switch part.FormName() {
		case "text":
			raw, err := io.ReadAll(io.LimitReader(part, captureLimit+1))
			_ = part.Close()
			if err != nil {
				return text, photo, kind, key, fmt.Errorf("reading what you said: %w", err)
			}
			text = said(string(raw))
		case "key":
			raw, err := io.ReadAll(io.LimitReader(part, 128))
			_ = part.Close()
			if err != nil {
				return text, photo, kind, key, fmt.Errorf("reading the capture's key: %w", err)
			}
			key = strings.TrimSpace(string(raw))
		case "photo":
			if part.FileName() == "" || opts.Photos == nil {
				_ = part.Close()
				continue
			}
			declared := part.Header.Get("Content-Type")
			if _, ok := squirrel.KnownKind(declared); !ok {
				_ = part.Close()
				return text, "", "", key, fmt.Errorf("%w: %q", errNotAPhotograph, declared)
			}
			photo, err = opts.Photos.Keep(part, declared)
			_ = part.Close()
			if err != nil {
				return text, "", "", key, fmt.Errorf("%w: %w", errNotAPhotograph, err)
			}
			kind = squirrel.PhotoKind(declared)
		default:
			_ = part.Close()
		}
	}
	return text, photo, kind, key, nil
}

func said(raw string) string {
	text := strings.TrimSpace(raw)
	if len(text) > captureLimit {
		text = text[:captureLimit]
	}
	return text
}
