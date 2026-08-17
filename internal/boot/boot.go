// Package boot wires the core to its transports. It is the only package that
// imports both, which is why Boot lives here rather than in squirrel — the
// core must never import a transport.
package boot

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
)

// presenceDebounce is not configuration — see PresenceOptions' own doc
// comment for what it does. It absorbs the rapid re-arrivals a flapping
// wifi/cellular handoff produces (Home Assistant fires on every one), and
// matches the value presence_test.go itself exercises. Unlike Delay, its
// value does not need to differ between production and the integration
// suite, so it stays a constant rather than joining config.PresenceDelay.
const presenceDebounce = 10 * time.Minute

// nudgeFunc matches Scheduler.Nudge's signature exactly, which is also what
// Applier.SetNudger expects.
type nudgeFunc = func(context.Context, time.Time, squirrel.NudgeReason) error

// nudgeRelay lets the presence route be mounted synchronously in Boot, on the
// same Server the Campfire webhook uses, before the scheduler that actually
// performs a nudge exists. The route has to be registered before Listen the
// same way every transport's route is; the scheduler can only be built once
// connectAndDrain reaches a live Postgres and learns the owner's person id,
// which may not have happened yet — or, with the database down, may not
// happen for a while. An arrival in that window is absorbed rather than
// queued, the same tradeoff MountPresence's own doc comment describes: a
// presence ping is not a thought, and losing one costs a nudge that the
// evening message still catches.
type nudgeRelay struct {
	fn atomic.Pointer[nudgeFunc]
}

func (r *nudgeRelay) set(f nudgeFunc) { r.fn.Store(&f) }

func (r *nudgeRelay) Nudge(ctx context.Context, now time.Time, why squirrel.NudgeReason) error {
	f := r.fn.Load()
	if f == nil {
		return nil
	}
	return (*f)(ctx, now, why)
}

type Squirrel struct {
	port    int
	server  *squirrel.Server
	store   *squirrel.Store
	stops   []func(context.Context) error
	cancel  context.CancelFunc
	drained chan struct{}
	// wg tracks background goroutines that touch the store outside the
	// drain's own goroutine — the digest scheduler, and each presence
	// arrival's delayed nudge — so Stop can join all of them before closing
	// the store. drained alone is not enough: it only covers connectAndDrain's
	// own goroutine, and both of these run concurrently with the drain loop
	// on a shared ctx, not nested inside it.
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

	// The relay is captured by OnArrive below and handed to connectAndDrain,
	// which fills it in once the scheduler exists. Mounting must happen here,
	// before Listen, the same as every transport's own route — not deferred
	// into connectAndDrain — so the route is live by the time Boot returns,
	// matching how the Campfire webhook is already live by then.
	nudge := &nudgeRelay{}
	if config.PresenceSecret != "" {
		squirrel.MountPresence(server, config.PresencePath, squirrel.PresenceOptions{
			Secret:   config.PresenceSecret,
			Debounce: presenceDebounce,
			Delay:    config.PresenceDelay,
			OnArrive: func() {
				if err := nudge.Nudge(loopCtx, time.Now(), squirrel.NudgeFromArrival); err != nil {
					slog.Error("nudge", "error", err)
				}
			},
			// Registers the delayed OnArrive call with s.wg instead of
			// leaving it a bare goroutine, the same guarantee wg already
			// gives the digest scheduler: Stop joins it before closing the
			// store, so an arrival caught mid-Delay or mid-Nudge at
			// shutdown finishes against a live pool rather than one Close
			// has already torn down. Add runs synchronously inside the HTTP
			// handler, before it returns — see presence.go's own comment on
			// Go — so it is guaranteed to have registered before
			// Shutdown's wait for in-flight handlers to return can be
			// satisfied, closing the same window a plain `go fn()` leaves
			// open between an arrival landing and Stop being called.
			Go: func(fn func()) {
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					fn()
				}()
			},
		})
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
		connectAndDrain(loopCtx, config, store, spool, transports, &s.wg, nudge)
	}()

	return s, nil
}

// connectAndDrain retries until Postgres answers, then drains until the
// context is cancelled. Nothing here blocks a capture being accepted.
func connectAndDrain(ctx context.Context, config squirrel.Config, store *squirrel.Store, spool *squirrel.Spool, transports []transport.Transport, wg *sync.WaitGroup, nudge *nudgeRelay) {
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
			scheduler := squirrel.NewScheduler(squirrel.SchedulerOptions{
				Store: store, Send: send, Chat: chat, PersonID: personID,
				ConversationID: config.Campfire.ConversationID,
				At:             config.EveningAt,
				Location:       config.DigestLocation,
				OnError:        func(err error) { slog.Error("digest", "error", err) },
			})

			// A capture can carry a nudge back on the same message, and an
			// arrival can trigger one through the presence route mounted
			// back in Boot — both go through this one Scheduler so the
			// once-a-day budget its unique index enforces is shared rather
			// than split across two independent claimants.
			applier.SetNudger(scheduler.Nudge)
			nudge.set(scheduler.Nudge)

			// Joined by Stop before the store closes: the scheduler runs on
			// the same ctx as the drain but is not nested inside it, so
			// draining alone does not wait for an in-flight digest send to
			// finish before the store is torn down.
			wg.Add(1)
			go func() {
				defer wg.Done()
				scheduler.Run(ctx)
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
