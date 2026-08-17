// Package boot wires the core to its transports. It is the only package that
// imports both, which is why Boot lives here rather than in squirrel — the
// core must never import a transport.
package boot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
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
	// wg tracks background goroutines started after the store opens — today
	// just the digest scheduler — so Stop can join them before closing the
	// store. drained alone is not enough: it only covers connectAndDrain's own
	// goroutine, and the scheduler runs concurrently with the drain loop on a
	// shared ctx, not nested inside it.
	wg sync.WaitGroup
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

	loopCtx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	transports := transportsFrom(config)

	for _, t := range transports {
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
		connectAndDrain(loopCtx, config, store, spool, transports, &s.wg)
	}()

	return s, nil
}

// connectAndDrain retries until Postgres answers, then drains until the
// context is cancelled. Nothing here blocks a capture being accepted.
func connectAndDrain(ctx context.Context, config squirrel.Config, store *squirrel.Store, spool *squirrel.Spool, transports []transport.Transport, wg *sync.WaitGroup) {
	var personID int64
	for {
		var err error
		if err = store.Migrate(ctx); err != nil {
			slog.Warn("database unavailable", "error", err, "retry_in", config.DrainInterval)
		} else if personID, err = store.SeedOwner(ctx, config.OwnerHandle, seedsFrom(config)); err != nil {
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

	// The transport's Send, or nil when no bot key is configured. A nil sender
	// means the applier stays quiet rather than crashing: phase 1's property
	// that this pod holds no Campfire credential is still a supported state.
	// chat is the richer surface alongside it — buttons on a message, and the
	// update that closes them — zero-valued the same way when it is absent.
	// Both the applier and the scheduler fall back to the plain-text sender
	// whenever chat.Send is nil, so wiring an empty Chat here is always safe.
	var send squirrel.Sender
	var chat squirrel.Chat
	for _, t := range transports {
		if t.Name == transport.CampfireName && t.Send != nil {
			send = squirrel.Sender(t.Send)
		}
		if t.Name == transport.CampfireName && t.Chat.Send != nil {
			chat = t.Chat
		}
	}

	var applier *squirrel.Applier
	if send != nil {
		applier = squirrel.NewApplier(store, send, chat, func(err error) {
			slog.Error("applying intent", "error", err)
		})

		if config.Campfire != nil {
			// Joined by Stop before the store closes: the scheduler runs on
			// the same ctx as the drain but is not nested inside it, so
			// draining alone does not wait for an in-flight digest send to
			// finish before the store is torn down.
			wg.Add(1)
			go func() {
				defer wg.Done()
				squirrel.NewScheduler(squirrel.SchedulerOptions{
					Store: store, Send: send, Chat: chat, PersonID: personID,
					ConversationID: config.Campfire.ConversationID,
					At:             config.DigestAt,
					Location:       config.DigestLocation,
					OnError:        func(err error) { slog.Error("digest", "error", err) },
				}).Run(ctx)
			}()
		}
	} else {
		slog.Warn("no sender configured; chores and the daily digest are inactive")
	}

	squirrel.NewDrain(squirrel.DrainOptions{
		Spool:    spool,
		Store:    store,
		Interval: config.DrainInterval,
		OnError:  func(err error) { slog.Error("drain error", "error", err) },
		OnUnknownIdentity: func(transport, senderID string) {
			slog.Warn("unknown identity", "transport", transport, "sender_id", senderID)
		},
		Applier: applier,
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
	s.wg.Wait()
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
