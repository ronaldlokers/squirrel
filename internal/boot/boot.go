// Package boot wires the core to its transports. It is the only package that
// imports both, which is why Boot lives here rather than in squirrel — the
// core must never import a transport.
package boot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ronaldlokers/squirrel/internal/coach"
	"github.com/ronaldlokers/squirrel/internal/squirrel"
	"github.com/ronaldlokers/squirrel/internal/transport"
	"github.com/ronaldlokers/squirrel/internal/web"
)

// presenceDebounce absorbs the rapid re-arrivals a flapping wifi/cellular
// handoff produces. Unlike Delay its value need not differ between production and
// the integration suite, so it stays a constant.
const presenceDebounce = 10 * time.Minute

// nudgeFunc matches Scheduler.Nudge's signature exactly, which is also what
// Applier.SetNudger expects.
type nudgeFunc = func(context.Context, time.Time, squirrel.NudgeReason) error

// nudgeRelay lets the presence route be mounted synchronously in Boot, before the
// scheduler that performs a nudge exists: routes must be registered before
// Listen, and the scheduler needs a live Postgres and the owner's person id.
//
// An arrival in that window is absorbed rather than queued — a presence ping is
// not a thought, and losing one costs a nudge the evening message still catches.
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
	port   int
	server *squirrel.Server
	store  *squirrel.Store
	stops  []func(context.Context) error
	// coach, budget and talk are built at boot and used by every surface that
	// asks a model. All three are safe to hold before Postgres answers:
	// nothing here touches the store until something is actually asked.
	coach coach.Coach
	// house is the small model on the cluster, or nil. Independent of the
	// coach on purpose: it needs no key and answers when the coach is absent.
	house  *coach.House
	budget coach.Budget
	// talk is the rolling window, shared by every surface so that two of them
	// cannot disagree about how long a conversation lasts.
	talk    *coach.Conversations
	cancel  context.CancelFunc
	drained chan struct{}
	// wg tracks background goroutines that touch the store outside the drain's own —
	// the scheduler, and each arrival's delayed nudge — so Stop can join them before
	// closing the store. drained alone covers only connectAndDrain's goroutine.
	wg sync.WaitGroup
}

func (s *Squirrel) Port() int { return s.port }

// Boot binds the HTTP server and starts transports BEFORE touching Postgres.
// Migrating first would mean a database outage during a restart produces a
// service that refuses connections, and Campfire does not retry.
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
			// Registered with s.wg rather than left a bare goroutine, so Stop joins it and an
			// arrival caught mid-Delay finishes against a live pool. Add runs synchronously
			// inside the handler, so it is registered before Shutdown's wait can be
			// satisfied.
			Go: func(fn func()) {
				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					fn()
				}()
			},
			// Ctx is loopCtx, which Stop cancels before joining s.wg. Without it an arrival
			// mid-Delay only wakes on its own timer — minutes — so a rollout blocks Stop past
			// the shutdown budget and the pod is SIGKILLed.
			Ctx: loopCtx,
		})
	} else {
		// MountPresence's "refusing to mount" log only fires when it is called, so a
		// mis-wired secretKeyRef produces no line at all — and a bot with no arrival
		// trigger looks exactly like one working normally.
		slog.Warn("no presence secret configured; the arrival trigger is inactive")
	}

	// Opened before Listen rather than after it, because the screen's routes
	// need the store to be mounted with and every route has to be registered
	// before the server accepts its first request. OpenStore does not connect
	// — pgxpool dials lazily — so this still does not put Postgres in front of
	// a capture being accepted.
	store, err := squirrel.OpenStore(loopCtx, squirrel.URLFor(config.Postgres))
	if err != nil {
		cancel()
		return nil, err
	}
	// Where the person is, for every question about which day it is. Threaded
	// rather than read off the process clock: a container's zone is an accident
	// of its deployment. See issue #148, and everything else here that takes a
	// location.
	store.In(config.DigestLocation)
	s.store = store

	// After the store, because the budget reads through it. A coach that is not
	// there must be an ordinary state at boot, not an error path.
	//
	// The owner id is declared here because the budget needs it too: the owner's
	// ceiling and a guest's are two numbers. It is zero until connectAndDrain fills
	// it in, and a lookup before then sees a guest, which is the safe way round.
	var webOwner atomic.Int64
	s.budget = budgetFor(config.Coach, store, webOwner.Load)
	s.coach = coachFor(config.Coach, s.budget, store)
	s.house = coach.NewHouse(config.Coach.HouseURL, config.Coach.HouseModel)
	if s.house != nil {
		slog.Info("there is a model in the house",
			"url", config.Coach.HouseURL, "model", config.Coach.HouseModel)
	}
	s.talk = coach.NewConversations()

	// Nowhere to keep a photograph is a supported state and the default.
	// Checked here rather than at the first press: the first photograph is the
	// worst moment to discover the volume is not there.
	var photos *squirrel.Photos
	if config.PhotoDir != "" {
		var err error
		if photos, err = squirrel.OpenPhotos(config.PhotoDir); err != nil {
			slog.Warn("no photo directory; the camera is not offered", "error", err)
			photos = nil
		} else {
			photos.Ceiling(int64(config.PhotoCeilingBytes))
			used, count := photos.Used()
			slog.Info("photographs are kept", "at", config.PhotoDir,
				"using", used, "photographs", count, "ceiling", config.PhotoCeilingBytes)
		}
	} else {
		slog.Info("no photo directory configured; the camera is not offered")
	}

	// The coach's seams for the screen, converted here because boot is the only
	// package that may know both shapes. All are nil-safe: with no coach the
	// four chips still answer and the ladder behind them is what shipped before
	// any of this existed.
	webAsk, webRecent, webRemember, webForget := coachWeb(s.coach, store, s.talk)
	makeSmaller := breaker(s.coach)
	split, splittable := splitter(s.coach)
	hold := interrupter(s.coach, store)
	// The weekly read of the record. Nil with no key, and the scheduler
	// checks the nil — see KnowingTick.
	learnBack := learner(s.coach)
	noticeBoard := noticer(s.coach)
	spent := spentFor(s.coach, s.budget)
	over := overFor(s.coach, s.budget)
	// The three tiers the box judges with, as one value a test can inspect.
	// See readingWiring.
	read := readingWiring(s.coach, store, s.house)

	// The way in, built once at boot and without touching the network:
	// discovery is lazy and retried, so an unreachable Authentik costs the
	// screen and never the boot. See NewAuthentik. What is refused here is
	// configuration that cannot come right on its own.
	var gate *web.Gate
	if config.OIDC.Ready() {
		var err error
		gate, err = web.NewAuthentik(ctx, web.Authentik{
			Issuer:        config.OIDC.Issuer,
			ClientID:      config.OIDC.ClientID,
			ClientSecret:  config.OIDC.ClientSecret,
			RedirectURL:   config.OIDC.RedirectURL,
			RequiredGroup: config.WebRequiredGroup,
		})
		if err != nil {
			cancel()
			return nil, fmt.Errorf("building the way in: %w", err)
		}
		slog.Info("the way in is open", "issuer", config.OIDC.Issuer,
			"group", config.WebRequiredGroup)
	}

	if config.WebIdentity != "" {
		if err := web.Mount(server, store, web.Options{
			Gate:          gate,
			RequiredGroup: config.WebRequiredGroup,
			// One cache in front of one table. The store is passed rather than
			// the pool: internal/web states what it needs and nothing more.
			Sessions: web.NewSessions(store),
			Login:    store.PersonForLogin,
			// What you are called and what you look like, which are not what
			// lets you in. See Options.RememberWho.
			RememberWho: store.RememberPerson,
			// Where the person is, the same location the store already took.
			Location: config.DigestLocation,
			// Empty unless the *whole* pair plus the contact is configured,
			// not merely the public half. A public key alone would mount the
			// subscribe route and let a browser sign up to a channel nothing
			// can send on: subscriptions stored, permission spent, and silence
			// — which is the one shape worse than never offering.
			PushKey: pushKeyFor(config.Push),
			// Assigned through a helper rather than straight from the pointer:
			// a nil *squirrel.Photos in an interface field is not a nil
			// interface, and the screen's own `opts.Photos == nil` would then
			// be false while every call panicked. It would draw the camera and
			// break on the press.
			Photos: photoStore(photos),
			Ask:    webAsk,
			// What the box is answered by. Nil with no key, and the nil is
			// what captureHandler checks — see reader.
			Reads: read.Reads,
			// The small model on the cluster, asked about everything typed. Nil falls through
			// to squirrel.LooksLikeAQuestion.
			//
			// This line was written, lost to a stray edit, and found only because a mutation
			// went looking. A field in an inline literal cannot be checked by a test, so
			// readingWiring exists and TestTheBoxIsGivenItsThreeTiers asserts all three.
			AskedAQuestion: read.AskedAQuestion,
			Recent:         webRecent,

			Remember: webRemember,
			Forget:   webForget,
			Smaller:  makeSmaller,

			Split:      split,
			Splittable: splittable,
			Spent:      spent,
		}); err != nil {
			cancel()
			return nil, fmt.Errorf("mounting the pile: %w", err)
		}
		slog.Info("the screen is mounted", "at", "/")
		// Chat can now say where it is. Only when it has been told: a link
		// built from a guess is a link that 404s.
		squirrel.SetScreenURL(config.WebURL)
	} else {
		// Same precedent as the presence warning above: a mis-wired
		// WEB_IDENTITY otherwise produces no log line at all, and a bot with
		// no screen looks exactly like one working normally.
		slog.Warn("no web identity configured; the pile screen is not mounted")
	}

	port, err := server.Listen(fmt.Sprintf(":%d", config.Port))
	if err != nil {
		cancel()
		return nil, err
	}
	s.port = port
	slog.Info("listening", "port", port)

	go func() {
		defer close(s.drained)
		connectAndDrain(loopCtx, draining{
			config: config, store: store, spool: spool, transports: transports,
			wg: &s.wg, nudge: nudge, webOwner: &webOwner,
			// false: chat has no cards to draw a place with. See Turn.CanOpen.
			ask:         coachChat(asker(s.coach, store, s.talk, false)),
			makeSmaller: makeSmaller,
			hold:        hold,
			learnBack:   learnBack,
			noticeBoard: noticeBoard,
			over:        over,
		})
	}()

	return s, nil
}

// Asker is the seam the core reaches a model through: a func of primitives,
// because internal/squirrel must not import internal/coach. Nil means there is
// no coach, which is an ordinary state and not a failure.
type Asker func(ctx context.Context, personID int64, kind, said, subject string) (
	text string, did []string, err error)

// draining is everything connectAndDrain needs. A struct rather than fourteen
// parameters, so a call site cannot pass two of them the wrong way round and
// still compile.
type draining struct {
	config     squirrel.Config
	store      *squirrel.Store
	spool      *squirrel.Spool
	transports []transport.Transport
	wg         *sync.WaitGroup
	nudge      *nudgeRelay
	webOwner   *atomic.Int64

	// The coach's seams, every one of them nil when there is no coach.
	ask         Asker
	makeSmaller squirrel.Breaker
	hold        squirrel.Interrupter
	learnBack   squirrel.Learner
	noticeBoard squirrel.Noticer
	over        func(context.Context, int64) bool
}

const migrationMaxBackoff = 30 * time.Second

// connectAndDrain retries until Postgres answers, then drains until the
// context is cancelled. Nothing here blocks a capture being accepted.
func connectAndDrain(ctx context.Context, w draining) {
	config, store, spool := w.config, w.store, w.spool
	var personID int64
	backoff := config.DrainInterval
	for {
		var err error
		if err = store.Migrate(ctx); err != nil {
			// Two very different failures used to share this line: "database unavailable"
			// sends you to look at Postgres, and a migration that will not apply is a bug in
			// the migration. They retry identically and are not diagnosed identically.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if squirrel.IsMigrationFailure(err) {
				slog.Error("a migration will not apply; the schema is at the last one that did and the drain is not running",
					"error", err, "retry_in", backoff,
					"what_to_do", "docs/running.md — a migration that will not apply")
			} else {
				slog.Warn("database unavailable", "error", err, "retry_in", backoff)
			}
		} else if personID, err = store.SeedOwner(ctx, config.OwnerHandle, seedsFrom(config)); err != nil {
			slog.Warn("seeding owner failed", "error", err, "retry_in", backoff)
		} else {
			break
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		backoff *= 2
		if backoff > migrationMaxBackoff {
			backoff = migrationMaxBackoff
		}
	}

	slog.Info("database ready")

	// The screen's routes have been live since Listen; this is the moment they
	// have something to read.
	w.webOwner.Store(personID)

	// The transport's Send, or nil when no bot key is configured, in which case the
	// applier stays quiet rather than crashing. chat is the richer surface alongside
	// it; both fall back to the plain sender when chat.Send is nil.
	var send squirrel.Sender
	var chat squirrel.Chat
	for _, t := range w.transports {
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
		// Where the person is, the same location the store already took.
		applier.In(config.DigestLocation)

		// Nil when there is no coach, and the nil carries meaning: chat does
		// not advertise `!coach` in help when there is nothing behind it. A
		// command that only ever answers "there is no coach" is worse than one
		// that was never offered.
		applier.SetCoach(w.ask)
		applier.SetBreaker(w.makeSmaller)
		applier.SetSpent(w.over)
		squirrel.SetCoachHere(w.ask != nil)

		if config.Campfire != nil {
			scheduler := squirrel.NewScheduler(schedulerOptionsFor(schedulerWiring{
				config: config, store: store, send: send, chat: chat,
				personID:       personID,
				conversationID: config.Campfire.ConversationID,
				// A veto, never a trigger: it is only ever asked about a
				// chore the rules already chose to raise.
				interrupt: w.hold,
				// Once a week, and only with a key. See KnowingTick.
				learn: w.learnBack,
				// Once a day, and only with a key. See NoticeTick.
				notice: w.noticeBoard,
			}))

			// A capture can carry a nudge back on the same message, and an
			// arrival can trigger one through the presence route mounted
			// back in Boot — both go through this one Scheduler so the
			// once-a-day budget its unique index enforces is shared rather
			// than split across two independent claimants.
			applier.SetNudger(scheduler.Nudge)
			w.nudge.set(scheduler.Nudge)

			// Joined by Stop before the store closes: the scheduler runs on
			// the same ctx as the drain but is not nested inside it, so
			// draining alone does not wait for an in-flight digest send to
			// finish before the store is torn down.
			w.wg.Add(1)
			go func() {
				defer w.wg.Done()
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
	// The screen is an identity like any other transport's, now that the slot spools:
	// the drain resolves a capture's owner from its sender, and one that resolves to
	// nobody lands as a row belonging to no one.
	var seeds []squirrel.IdentitySeed
	if c.WebIdentity != "" {
		seeds = append(seeds, squirrel.IdentitySeed{
			Transport: squirrel.ScreenTransport, ExternalID: c.WebIdentity,
		})
	}
	// The owner's sub, so logging in resolves to the person who already owns the pile
	// rather than making a second one.
	//
	// Both transports: the drain resolves a spooled capture's owner from its sender,
	// so the owner needs a `screen` identity under the sub as well as under whatever
	// the header used to say.
	if c.WebOwnerSub != "" {
		seeds = append(seeds,
			squirrel.IdentitySeed{Transport: squirrel.OIDCTransport, ExternalID: c.WebOwnerSub},
			squirrel.IdentitySeed{Transport: squirrel.ScreenTransport, ExternalID: c.WebOwnerSub})
	}

	if c.Campfire == nil {
		return seeds
	}
	return append(seeds, squirrel.IdentitySeed{
		Transport: transport.CampfireName, ExternalID: c.Campfire.SenderID,
	})
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// schedulerWiring is everything the scheduler is given, named so the giving can
// be tested.
type schedulerWiring struct {
	// config whole, rather than the four values picked out of it, and that is
	// the point rather than brevity. Every field this function needs from
	// configuration it now reads itself, so there is no argument a caller can
	// pass the wrong thing for and none it can forget. Handing it `config.Push`
	// separately compiled just as well when it was handed an empty one.
	config   squirrel.Config
	store    *squirrel.Store
	send     squirrel.Sender
	chat     squirrel.Chat
	personID int64
	// conversationID stays an argument: Boot has already established that
	// config.Campfire is non-nil by the time it gets here, and re-deriving it
	// would mean this function handling a case its only caller cannot produce.
	conversationID string
	interrupt      squirrel.Interrupter
	// notice is the daily read of the board. Nil with no coach.
	notice squirrel.Noticer
	// learn is the weekly read of the record. Nil with no coach, which is the
	// state the product was in for a month and works.
	learn squirrel.Learner
}

// schedulerOptionsFor is the struct literal boot wrote inline, lifted out so what
// it carries can be asserted.
//
// One field was missing from that literal and nothing could see it: `Push` was
// never set, so MomentTick's nil check was false on every tick since the feature
// shipped. Go does not warn about an unused package-level function, and the
// symptom is a notification that does not arrive.
func schedulerOptionsFor(w schedulerWiring) squirrel.SchedulerOptions {
	return squirrel.SchedulerOptions{
		Store: w.store, Send: w.send, Chat: w.chat, PersonID: w.personID,
		ConversationID: w.conversationID,
		At:             w.config.EveningAt,
		Location:       w.config.DigestLocation,
		OnError:        func(err error) { slog.Error("digest", "error", err) },
		Interrupt:      w.interrupt,
		Learn:          w.learn,
		Notice:         w.notice,
		Push:           pusher(w.config.Push, w.store),
	}
}

// subscriptions is the half of the store this needs, named so the fan-out can
// be exercised without a database. The concrete store satisfies it.
type subscriptions interface {
	LiveSubscriptions(ctx context.Context, personID int64) ([]squirrel.Subscription, error)
	SubscriptionGone(ctx context.Context, id int64, at time.Time) error
	RecordSaid(ctx context.Context, personID int64, p squirrel.Push, at time.Time) error
}

// pusher builds the fast channel, or nil when there is no VAPID pair — a
// supported state, because every message this carries still reaches the room.
//
// Every failure here is swallowed by the caller: none of them may turn a message
// that has already arrived somewhere into an error.
func pusher(cfg squirrel.PushConfig, store subscriptions) squirrel.Pusher {
	if !cfg.Enabled() {
		slog.Warn("no push keys configured; only the room is told about leaving")
		return nil
	}
	// Its own client with a short timeout rather than the default one: a push
	// service that hangs must not hold the scheduler's minute open, and a
	// warning about leaving is worthless by the time a default timeout would
	// have expired anyway.
	client := &http.Client{Timeout: 10 * time.Second}

	return func(ctx context.Context, personID int64, p squirrel.Push) error {
		subs, err := store.LiveSubscriptions(ctx, personID)
		if err != nil {
			return err
		}
		// Saying there is nobody is the whole point of this line. This path used to log
		// only failures, so a send to an empty list and a send that worked produced
		// identical logs — which is how it ran in production for weeks with zero
		// subscribers and no way to notice.
		if len(subs) == 0 {
			slog.Warn("nobody to push to; only the room was told", "person_id", personID)
			return nil
		}
		slog.Info("pushing", "subscriptions", len(subs))
		took := 0
		for _, sub := range subs {
			gone, err := squirrel.SendPush(ctx, client, cfg, sub, p)
			if err != nil {
				slog.Error("push", "endpoint", host(sub.Endpoint), "error", err)
				continue
			}
			if gone {
				// The browser is gone for good. Retrying it forever makes every
				// later send slower for nothing.
				slog.Warn("a browser is gone; retiring its subscription",
					"endpoint", host(sub.Endpoint))
				if err := store.SubscriptionGone(ctx, sub.ID, time.Now()); err != nil {
					slog.Error("retiring a push subscription", "error", err)
				}
				continue
			}
			// The push service took it. That is as far as this side can see:
			// what the phone then does with it is not observable from here, and
			// saying so is better than a line that implies otherwise.
			slog.Info("pushed", "endpoint", host(sub.Endpoint))
			took++
		}
		// Kept once, and only when a push service took it. A row written when
		// every endpoint refused would be the app telling you it said something
		// it did not say, which is worse than a list with a gap in it.
		if took > 0 {
			if err := store.RecordSaid(ctx, personID, p, time.Now()); err != nil {
				slog.Error("keeping what was said", "error", err)
			}
		}
		return nil
	}
}

// host is the push service's name without the path, which is the part of an
// endpoint that identifies a browser. A whole endpoint in a log line is a
// credential of sorts — anyone holding it can push to that browser — and the
// same reasoning already keeps the bot key and the presence token out of logs.
func host(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil {
		return "?"
	}
	return u.Host
}

// pushKeyFor is the public key, and only when sending would actually work.
//
// Enabled() wants all three settings. Handing the screen a public key while
// the private half is missing would draw a button, spend a permission prompt,
// and store a subscription that nothing will ever push to.
func pushKeyFor(cfg squirrel.PushConfig) string {
	if !cfg.Enabled() {
		return ""
	}
	return cfg.PublicKey
}

// photoStore keeps a nil pointer from becoming a non-nil interface.
//
// The classic Go trap, and it would land exactly where it hurts: the screen
// asks `opts.Photos == nil` to decide whether to offer a camera, and a typed
// nil answers false. It would draw the button and panic on the press.
func photoStore(p *squirrel.Photos) web.Photos {
	if p == nil {
		return nil
	}
	return p
}
