package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	// distroless/static carries no tzdata, and LoadLocation would silently fall
	// back to UTC — shifting the digest by an hour or two depending on the
	// season, which nobody would notice for months.
	_ "time/tzdata"

	"github.com/ronaldlokers/squirrel/internal/boot"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	squirrel, err := boot.Boot(ctx, environ())
	if err != nil {
		slog.Error("boot failed", "error", err)
		os.Exit(1)
	}

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := squirrel.Stop(shutdownCtx); err != nil {
		slog.Error("shutdown failed", "error", err)
		os.Exit(1)
	}
}

func environ() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		if key, value, found := strings.Cut(entry, "="); found {
			env[key] = value
		}
	}
	return env
}
