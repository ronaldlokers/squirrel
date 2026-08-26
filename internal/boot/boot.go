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
	talk *coach.Conversations
	// offers holds the last decision per person, so an idle reopen costs
	// nothing and does not read as the product changing its mind.
	offers  *coach.Offers
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
			// Ctx is loopCtx, the same context Stop cancels (via s.cancel,
			// see below) before it joins s.wg. Without this, an arrival
			// caught mid-Delay only ever wakes on its own timer — up to
			// PresenceDelay, a couple of minutes in production — so a
			// rollout in that window blocks Stop past main's 15s shutdown
			// budget and the default 30s grace period, and the pod is
			// SIGKILLed rather than stopped cleanly. Threading loopCtx here
			// lets cancellation wake it immediately instead.
			Ctx: loopCtx,
		})
	} else {
		// MountPresence's own "refusing to mount" log only fires when it is
		// actually called — a mis-wired secretKeyRef that leaves
		// PRESENCE_SECRET empty otherwise produces no log line at all, and a
		// bot with no arrival trigger looks exactly like one working
		// normally, since the evening message still runs on schedule. Same
		// precedent as the "no sender configured" warning below for a
		// missing bot key.
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
	// Where the person is, for every question this store answers about which
	// day it is. The same location the scheduler already takes — see #148.
	store.In(config.DigestLocation)
	s.store = store

	// After the store, because the budget reads through it; before the screen,
	// because the screen will ask for the coach in the phase that adds the
	// sheet. Neither one connecting to anything yet is the point: a coach that
	// is not there must be an ordinary state at boot, not an error path.
	// Declared here rather than beside the screen it feeds because the budget
	// needs it too: the owner's monthly ceiling and a guest's are two numbers,
	// and telling them apart means knowing who the owner is. It is zero until
	// connectAndDrain fills it in, and a ceiling lookup that runs before then
	// sees a guest — which is the safe way round.
	var webOwner atomic.Int64
	s.budget = budgetFor(config.Coach, store, webOwner.Load)
	s.coach = coachFor(config.Coach, s.budget, store)
	s.house = coach.NewHouse(config.Coach.HouseURL, config.Coach.HouseModel)
	if s.house != nil {
		slog.Info("there is a model in the house",
			"url", config.Coach.HouseURL, "model", config.Coach.HouseModel)
	}
	s.offers = coach.NewOffers()
	s.talk = coach.NewConversations()

	// The owner is filled in by connectAndDrain once Postgres answers, which
	// may be a while after boot or never. Same shape and same reason as
	// nudgeRelay above: the route is registered here, before Listen, and the
	// thing it needs arrives later. Until it does, the screen answers 503 —
	// the pile is unreadable because its memory is unreachable, which is what
	// that status already means on this screen.
	// Nowhere to keep a photograph is a supported state and the default. It is
	// checked at boot rather than at the first press, for the reason the spool
	// is: the first photograph is the worst moment to discover the volume is
	// not there.
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

	// The coach's three seams for the screen, converted here because boot is
	// the only package that may know both shapes. All three are nil-safe: with
	// no coach the acorn still opens, the four chips still answer, and the
	// ladder behind them is what shipped before any of this existed.
	webAsk, webRecent, webRemember, webForget := coachWeb(s.coach, store, s.talk)
	// One decider, shared by the screen and the chat, so both see the same
	// cached answer and asking twice costs once.
	// The decision, and the way to drop it. One call, because they are only
	// correct against the same cache — see deciding.
	decide, forgetOffer := deciding(s.coach, s.offers)
	makeSmaller := breaker(s.coach)
	split, splittable := splitter(s.coach)
	hold := interrupter(s.coach, store)
	// The weekly read of the record. Nil with no key, and the scheduler
	// checks the nil — see KnowingTick.
	learnBack := learner(s.coach)
	spent := spentFor(s.coach, s.budget)
	over := overFor(s.coach, s.budget)
	// The three tiers the box judges with, as one value a test can inspect.
	// See readingWiring.
	read := readingWiring(s.coach, store, s.house)

	// The way in, built once at boot. Discovery is a network call to
	// Authentik, and a Squirrel that cannot find its own way in is not a
	// working Squirrel — so this fails the boot rather than mounting a screen
	// nobody can reach.
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
			// The same location the scheduler's quiet hours and evening
			// message already take. Threaded rather than read off the process
			// clock — see issue #148.
			Location: config.DigestLocation,
			// Empty unless the *whole* pair plus the contact is configured,
			// not merely the public half. A public key alone would mount the
			// subscribe route and let a browser sign up to a channel nothing
			// can send on: subscriptions stored, permission spent, and silence
			// — which is the one shape worse than never offering.
			PushKey: pushKeyFor(config.Push),
			// The same spool the room's captures go through, so there is one
			// durable path into the pile rather than two to keep in step.
			Spool: spool,
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
			// The small model on the cluster, asked about everything typed.
			// Nil falls through to squirrel.LooksLikeAQuestion, which needs
			// nothing running — see whatBuddyMakesOfIt.
			//
			// This line was written, lost to a stray edit, and only found
			// because a mutation went looking for it. A field in an inline
			// literal cannot be checked by a test, which is the whole of what
			// `Push` cost three releases — so readingWiring exists and
			// TestTheBoxIsGivenItsThreeTiers asserts all three.
			AskedAQuestion: read.AskedAQuestion,
			Recent:         webRecent,

			Remember:    webRemember,
			Forget:      webForget,
			Decide:      decide,
			ForgetOffer: forgetOffer,
			Smaller:     makeSmaller,

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
		connectAndDrain(loopCtx, config, store, spool, transports, &s.wg, nudge, &webOwner,
			// false: chat has no cards to draw a place with. See Turn.CanOpen.
			coachChat(asker(s.coach, store, s.talk, false)), decide, makeSmaller, hold, learnBack, over)
	}()

	return s, nil
}

// Asker is the seam the core reaches a model through: a func of primitives,
// because internal/squirrel must not import internal/coach. Nil means there is
// no coach, which is an ordinary state and not a failure.
type Asker func(ctx context.Context, personID int64, kind, said, subject string) (
	text string, did []string, err error)

// connectAndDrain retries until Postgres answers, then drains until the
// context is cancelled. Nothing here blocks a capture being accepted.
func connectAndDrain(ctx context.Context, config squirrel.Config, store *squirrel.Store, spool *squirrel.Spool, transports []transport.Transport, wg *sync.WaitGroup, nudge *nudgeRelay, webOwner *atomic.Int64, ask Asker, decide squirrel.Decider, makeSmaller squirrel.Breaker,
	hold squirrel.Interrupter, learnBack squirrel.Learner, over func(context.Context, int64) bool) {
	var personID int64
	for {
		var err error
		if err = store.Migrate(ctx); err != nil {
			// Two very different failures used to share this line, and the
			// wrong one of them was the default reading: "database
			// unavailable" sends you to look at Postgres, and a migration that
			// will not apply is a bug in the migration with Postgres answering
			// perfectly well. They retry identically and for the same reason —
			// nothing here may block a capture being accepted — but they do
			// not get diagnosed identically, so they no longer read the same.
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return
			}
			if squirrel.IsMigrationFailure(err) {
				slog.Error("a migration will not apply; the schema is at the last one that did and the drain is not running",
					"error", err, "retry_in", config.DrainInterval,
					"what_to_do", "docs/running.md — a migration that will not apply")
			} else {
				slog.Warn("database unavailable", "error", err, "retry_in", config.DrainInterval)
			}
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

	// The screen's routes have been live since Listen; this is the moment they
	// have something to read.
	webOwner.Store(personID)

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
		// Where the person is. The same location the scheduler's quiet hours
		// and evening message take, threaded rather than read off the process
		// clock — see issue #148.
		applier.In(config.DigestLocation)

		// Nil when there is no coach, and the nil carries meaning: chat does
		// not advertise `!coach` in help when there is nothing behind it. A
		// command that only ever answers "there is no coach" is worse than one
		// that was never offered.
		applier.SetCoach(ask)
		applier.SetDecider(decide)
		applier.SetBreaker(makeSmaller)
		applier.SetSpent(over)
		squirrel.SetCoachHere(ask != nil)

		if config.Campfire != nil {
			scheduler := squirrel.NewScheduler(schedulerOptionsFor(schedulerWiring{
				config: config, store: store, send: send, chat: chat,
				personID:       personID,
				conversationID: config.Campfire.ConversationID,
				// A veto, never a trigger: it is only ever asked about a
				// chore the rules already chose to raise.
				interrupt: hold,
				// Once a week, and only with a key. See KnowingTick.
				learn: learnBack,
			}))

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
	// The screen is an identity like any other transport's.
	//
	// It has to be, now that the slot spools rather than writing straight to
	// Postgres: the drain resolves a capture's owner from its sender, and a
	// capture whose sender resolves to nobody lands as a row belonging to no
	// one. Seeding this is what keeps a note typed on the screen yours.
	var seeds []squirrel.IdentitySeed
	if c.WebIdentity != "" {
		seeds = append(seeds, squirrel.IdentitySeed{
			Transport: squirrel.ScreenTransport, ExternalID: c.WebIdentity,
		})
	}
	// The owner's sub, so that logging in resolves to the person who already
	// owns the pile rather than making a second one beside it.
	//
	// Both transports, and the screen one is why WEB_IDENTITY is still here:
	// the drain resolves a spooled capture's owner from its sender string, so
	// the owner needs a `screen` identity under the sub as well as under
	// whatever the header used to say. Two rows rather than one is what keeps
	// a note typed before the deploy and a note typed after it the same
	// person's.
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
	// learn is the weekly read of the record. Nil with no coach, which is the
	// state the product was in for a month and works.
	learn squirrel.Learner
}

// schedulerOptionsFor is the struct literal boot used to write inline, lifted
// out so that what it carries can be asserted.
//
// It exists because one field was missing from that literal and nothing could
// see it. `Push` was never set, so `MomentTick`'s own `if s.opts.Push != nil`
// was false on every tick since the feature shipped, and `pusher` — written,
// commented and unit-tested — was never called by anything but its test. Go
// does not warn about an unused package-level function, and the symptom is a
// notification that does not arrive, which is indistinguishable from every
// other reason a notification might not arrive.
//
// A field set in an inline literal cannot be checked by a test. A field set
// here can, and pushwired_test.go does.
func schedulerOptionsFor(w schedulerWiring) squirrel.SchedulerOptions {
	return squirrel.SchedulerOptions{
		Store: w.store, Send: w.send, Chat: w.chat, PersonID: w.personID,
		ConversationID: w.conversationID,
		At:             w.config.EveningAt,
		Location:       w.config.DigestLocation,
		OnError:        func(err error) { slog.Error("digest", "error", err) },
		Interrupt:      w.interrupt,
		Learn:          w.learn,
		Push:           pusher(w.config.Push, w.store),
	}
}

// subscriptions is the half of the store this needs, named so the fan-out can
// be exercised without a database. The concrete store satisfies it.
type subscriptions interface {
	LiveSubscriptions(ctx context.Context, personID int64) ([]squirrel.Subscription, error)
	SubscriptionGone(ctx context.Context, id int64, at time.Time) error
}

// pusher builds the fast channel, or nil.
//
// Nil when there is no VAPID pair, which is a supported state rather than a
// degraded one: every message this would carry still reaches the room, and the
// room is the channel that always works.
//
// Every failure here is swallowed by the caller. A push service being slow, a
// browser having revoked its subscription, a laptop that is closed — none of
// those may turn a message that has already arrived somewhere into an error.
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
		// Saying there is nobody is the whole point of this line.
		//
		// This path used to say nothing at all: no line when it sent, none for
		// how many browsers it found, none on success. Only a failure spoke. So
		// a send to an empty list and a send that worked produced identical
		// logs — which is how this feature ran in production for weeks with
		// zero subscribers and no way to notice, and why finding that out took
		// four rounds of guessing.
		//
		// A channel that cannot report having no listeners cannot be trusted to
		// report anything.
		if len(subs) == 0 {
			slog.Warn("nobody to push to; only the room was told", "person_id", personID)
			return nil
		}
		slog.Info("pushing", "subscriptions", len(subs))
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
