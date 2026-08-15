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

type DrainOptions struct {
	Spool      *Spool
	Store      *Store
	Interval   time.Duration
	MaxBackoff time.Duration
	OnError    func(error)
	// OnUnknownIdentity is not an error channel. The row still lands; nobody
	// knows whose it is yet.
	OnUnknownIdentity func(transport, senderID string)
}

type Drain struct {
	opts DrainOptions
}

func NewDrain(o DrainOptions) *Drain {
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

	if err := d.opts.Store.InsertItem(ctx, Item{
		Transport:      capture.Transport,
		ExternalID:     capture.ExternalID,
		ConversationID: capture.ConversationID,
		SenderID:       capture.SenderID,
		PersonID:       personID,
		RawText:        capture.Text,
		Payload:        capture.Payload,
		ReceivedAt:     capture.ReceivedAt,
	}); err != nil {
		return d.handle(name, err)
	}

	if err := d.opts.Spool.Remove(name); err != nil {
		// The row landed. Leaving the file means one redelivery, which the
		// ON CONFLICT makes harmless.
		d.report(err)
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

		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
