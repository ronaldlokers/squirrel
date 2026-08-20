package squirrel

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type DrainResult struct {
	Inserted    int
	Quarantined int
	Deferred    int
}

// DrainStore is the subset of Store the drain depends on. *Store satisfies it
// structurally, so production callers pass one unchanged; tests can
// substitute a smaller implementation — e.g. one that fails a fixed number of
// times before delegating to a real Store — to force a deferred pass without
// needing an actually unreachable Postgres for a whole test.
type DrainStore interface {
	ResolvePerson(ctx context.Context, transport string, externalID *string) (*int64, error)
	// InsertItem reports whether it actually inserted a row, as opposed to
	// silently absorbing a redelivery via ON CONFLICT DO NOTHING. The drain
	// depends on that distinction to apply an intent at most once.
	InsertItem(ctx context.Context, i Item) (bool, error)
}

type DrainOptions struct {
	Spool      *Spool
	Store      DrainStore
	Interval   time.Duration
	MaxBackoff time.Duration
	OnError    func(error)
	// OnUnknownIdentity is not an error channel. The row still lands; nobody
	// knows whose it is yet.
	OnUnknownIdentity func(transport, senderID string)
	// OnWait reports the interval Run decided to wait before its next pass,
	// every pass. It exists for one reason: the backoff is the only part of
	// this loop whose behaviour is a *duration*, and a test that measures it
	// by timing real ticks is measuring the machine as much as the code.
	//
	// Reporting the decision makes the test assert what Run chose rather than
	// what a loaded CI runner delivered — which is both the thing worth
	// checking and the only version of it that does not flake. Nil in
	// production; nothing outside a test sets it.
	OnWait func(time.Duration)
	// Applier runs after a capture lands. Nil keeps phase-1 behaviour.
	Applier *Applier
}

type Drain struct {
	opts DrainOptions
}

// defaultInterval is used when Interval is left at its zero value. Run's
// backoff starts at Interval and only ever grows by doubling it, so a zero
// Interval never grows past zero either — every tick fires immediately,
// hammering Postgres and the spool directory as fast as the CPU allows.
const defaultInterval = time.Second

func NewDrain(o DrainOptions) *Drain {
	if o.Interval <= 0 {
		o.Interval = defaultInterval
	}
	if o.MaxBackoff == 0 {
		o.MaxBackoff = 30 * time.Second
	}
	return &Drain{opts: o}
}

// permanent means retrying cannot help: the file is not readable as a capture,
// or the row violates a constraint that will still be violated next time.
//
// Everything else — connection refused, a failover in progress, an error
// nobody anticipated — is transient, because deferring costs a second and
// quarantining a good capture costs the thought.
func permanent(err error) bool {
	if errors.Is(err, ErrMalformedSpoolFile) {
		return true
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return strings.HasPrefix(pgErr.Code, "22") || strings.HasPrefix(pgErr.Code, "23")
	}
	return false
}

func (d *Drain) report(err error) {
	if d.opts.OnError != nil {
		safely(func() { d.opts.OnError(err) })
	}
}

func (d *Drain) Once(ctx context.Context) DrainResult {
	var result DrainResult

	names, err := d.opts.Spool.List()
	if err != nil {
		d.report(err)
		return DrainResult{Deferred: 1}
	}

	for _, name := range names {
		if ctx.Err() != nil {
			return result
		}
		switch d.one(ctx, name) {
		case Stored:
			result.Inserted++
		case Failed:
			result.Deferred++
		case Ignored:
			result.Quarantined++
		}
	}
	return result
}

// one reuses Outcome rather than inventing a second vocabulary: Stored means
// the row landed and the file is gone, Ignored means quarantined, Failed means
// deferred for the next pass.
func (d *Drain) one(ctx context.Context, name string) Outcome {
	capture, err := d.opts.Spool.Read(name)
	if err != nil {
		return d.handle(name, err)
	}

	personID, err := d.opts.Store.ResolvePerson(ctx, capture.Transport, capture.SenderID)
	if err != nil {
		return d.handle(name, err)
	}
	if personID == nil && capture.SenderID != nil && d.opts.OnUnknownIdentity != nil {
		safely(func() { d.opts.OnUnknownIdentity(capture.Transport, *capture.SenderID) })
	}

	item := Item{
		Transport:      capture.Transport,
		ExternalID:     capture.ExternalID,
		ConversationID: capture.ConversationID,
		SenderID:       capture.SenderID,
		PersonID:       personID,
		RawText:        capture.Text,
		Payload:        capture.Payload,
		ReceivedAt:     capture.ReceivedAt,
	}
	inserted, err := d.opts.Store.InsertItem(ctx, item)
	if err != nil {
		return d.handle(name, err)
	}

	if err := d.opts.Spool.Remove(name); err != nil {
		// The row landed, or its redelivery was silently absorbed by ON
		// CONFLICT. Leaving the file means the same message is read again on
		// a later pass; that redelivery is harmless to the row either way,
		// and inserted below keeps it harmless to the Applier too.
		d.report(err)
	}

	// inserted is false when ON CONFLICT DO NOTHING absorbed a redelivery —
	// the row already existed, so this message was already applied once.
	// Gating on it here is what keeps a redelivered "done" from recording a
	// second completion or sending a second reply.
	//
	// A capture with no conversation is not applied at all, and the rule is
	// about what the applier is *for*: every branch of it ends in something
	// said back, so with nowhere to say it there is nothing to run. The
	// screen's slot is the case that made this matter — it spools now, the
	// same as the room, and "anything you type is a note" has always been the
	// whole of what it does. Running Match over it would quietly turn the slot
	// into a command line and try to answer into a conversation that does not
	// exist.
	if inserted && d.opts.Applier != nil && item.ConversationID != nil {
		if err := d.opts.Applier.Apply(ctx, item, personID); err != nil {
			// The row landed and the file is gone. A failed reply is a lost
			// answer, never a lost thought, so it is reported and not retried.
			d.report(err)
		}
	}
	return Stored
}

func (d *Drain) handle(name string, err error) Outcome {
	d.report(err)
	if !permanent(err) {
		return Failed
	}
	// Never deleted. A file that cannot be inserted must not spin forever, and
	// it must not disappear either.
	if qErr := d.opts.Spool.Quarantine(name); qErr != nil {
		d.report(qErr)
		return Failed
	}
	return Ignored
}

// Run drains on an interval until the context is cancelled, backing off while
// anything is deferred so a Postgres outage does not become a hammering loop.
func (d *Drain) Run(ctx context.Context) {
	backoff := d.opts.Interval

	for {
		result := d.Once(ctx)
		if result.Deferred > 0 {
			backoff *= 2
			if backoff > d.opts.MaxBackoff {
				backoff = d.opts.MaxBackoff
			}
			slog.Warn("drain deferred", "files", result.Deferred, "retry_in", backoff)
		} else {
			backoff = d.opts.Interval
		}

		if d.opts.OnWait != nil {
			d.opts.OnWait(backoff)
		}

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
