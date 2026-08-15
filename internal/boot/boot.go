// Package boot wires the core to its transports. It is the only package that
// imports both, which is why Boot lives here rather than in squirrel — the
// core must never import a transport.
package boot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
)

type Squirrel struct {
	port    int
	server  *squirrel.Server
	store   *squirrel.Store
	stops   []func(context.Context) error
	cancel  context.CancelFunc
	drained chan struct{}
}

func (s *Squirrel) Port() int { return s.port }

// Boot binds the HTTP server and starts transports BEFORE touching Postgres.
//
// Migrating first would mean a database outage during a pod restart produces a
// service that refuses connections, and every message sent in that window is
// gone — Campfire does not retry. That is the failure the spool exists to
// prevent, so it must not be reintroduced at boot.
func Boot(ctx context.Context, env map[string]string) (*Squirrel, error) {
	config, err := squirrel.LoadConfig(env)
	if err != nil {
		return nil, err
	}

	spool, err := squirrel.OpenSpool(config.SpoolDir)
	if err != nil {
		return nil, err
	}
	if swept, err := spool.Sweep(); err != nil {
		return nil, err
	} else if swept > 0 {
		slog.Info("swept partial spool files", "files", swept)
	}

	sink := squirrel.NewSink(spool, allowsFrom(config), squirrel.SinkHooks{
		OnError: func(err error) { slog.Error("spool write failed", "error", err) },
		// Silent in the room, visible here. Being added to a room Squirrel
		// does not belong in should show up somewhere.
		OnIgnored: func(c squirrel.Capture) {
			slog.Warn("capture ignored", "transport", c.Transport,
				"conversation_id", deref(c.ConversationID), "sender_id", deref(c.SenderID))
		},
	})

	server := squirrel.NewServer(spool)
	s := &Squirrel{server: server, drained: make(chan struct{})}

	loopCtx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel

	for _, t := range transportsFrom(config) {
		stop, err := t.Start(loopCtx, sink, server)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("starting transport %s: %w", t.Name, err)
		}
		s.stops = append(s.stops, stop)
		slog.Info("transport started", "transport", t.Name, "sends", t.Send != nil)
	}

	port, err := server.Listen(fmt.Sprintf(":%d", config.Port))
	if err != nil {
		cancel()
		return nil, err
	}
	s.port = port
	slog.Info("listening", "port", port)

	store, err := squirrel.OpenStore(loopCtx, squirrel.URLFor(config.Postgres))
	if err != nil {
		cancel()
		return nil, err
	}
	s.store = store

	go func() {
		defer close(s.drained)
		connectAndDrain(loopCtx, config, store, spool)
	}()

	return s, nil
}

// connectAndDrain retries until Postgres answers, then drains until the
// context is cancelled. Nothing here blocks a capture being accepted.
func connectAndDrain(ctx context.Context, config squirrel.Config, store *squirrel.Store, spool *squirrel.Spool) {
	for {
		if err := store.Migrate(ctx); err != nil {
			slog.Warn("database unavailable", "error", err, "retry_in", config.DrainInterval)
		} else if _, err := store.SeedOwner(ctx, config.OwnerHandle, seedsFrom(config)); err != nil {
			slog.Warn("seeding owner failed", "error", err, "retry_in", config.DrainInterval)
		} else {
			break
		}

		timer := time.NewTimer(config.DrainInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}

	slog.Info("database ready")
	squirrel.NewDrain(squirrel.DrainOptions{
		Spool:    spool,
		Store:    store,
		Interval: config.DrainInterval,
		OnError:  func(err error) { slog.Error("drain error", "error", err) },
		OnUnknownIdentity: func(transport, senderID string) {
			slog.Warn("unknown identity", "transport", transport, "sender_id", senderID)
		},
	}).Run(ctx)
}

// Stop drains in-flight requests before closing anything, so a rollout does
// not sever a webhook Campfire will never retry.
func (s *Squirrel) Stop(ctx context.Context) error {
	err := s.server.Shutdown(ctx)
	for _, stop := range s.stops {
		if stopErr := stop(ctx); err == nil {
			err = stopErr
		}
	}
	s.cancel()
	<-s.drained
	if s.store != nil {
		s.store.Close()
	}
	return err
}

func transportsFrom(c squirrel.Config) []transport.Transport {
	built := []transport.Transport{}
	if c.Campfire != nil {
		built = append(built, transport.NewCampfire(*c.Campfire))
	}
	return built
}

func allowsFrom(c squirrel.Config) []squirrel.Allow {
	if c.Campfire == nil {
		return nil
	}
	return []squirrel.Allow{{
		Transport:      transport.CampfireName,
		ConversationID: c.Campfire.ConversationID,
		SenderID:       c.Campfire.SenderID,
	}}
}

func seedsFrom(c squirrel.Config) []squirrel.IdentitySeed {
	if c.Campfire == nil {
		return nil
	}
	return []squirrel.IdentitySeed{{
		Transport: transport.CampfireName, ExternalID: c.Campfire.SenderID,
	}}
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
