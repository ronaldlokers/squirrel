//go:build integration

package squirrel_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// Every reason Buddy goes quiet gives the same behaviour on purpose — the
// picker chooses, the ladder answers, Rule 10 holds. One of them is not "try
// again in a minute", though: a spent month is spent until the first, and
// typing the same thing four more times at eleven at night is the one outcome
// this can spare you.
//
// The screen has shown the figure in the sheet's lid all along. A session that
// lives in the room never sees it.
func TestChatSaysWhenTheMonthIsSpent(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetCoach(func(context.Context, int64, string, string, string) (string, []string, error) {
		return "", nil, errors.New("no coach available")
	})
	a.SetSpent(func(context.Context, int64) bool { return true })

	item := itemOf("!buddy everything at once")
	item.PersonID = squirrel.Ptr(p)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "done for this month",
		"the room could not tell a spent month from a bad minute")

	for _, money := range []string{"€", "0.0", "spent", "ceiling", "budget"} {
		require.NotContains(t, (*got)[0].text, money,
			"the room was handed a number about money")
	}
}

// And an ordinary failure still says what it always said. A month that is fine
// must not be reported as spent, or the message means nothing.
func TestAnOrdinaryFailureStillSaysWhatItSaid(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetCoach(func(context.Context, int64, string, string, string) (string, []string, error) {
		return "", nil, errors.New("no coach available")
	})
	a.SetSpent(func(context.Context, int64) bool { return false })

	item := itemOf("!buddy everything at once")
	item.PersonID = squirrel.Ptr(p)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Len(t, *got, 1)
	require.Contains(t, (*got)[0].text, "Nothing useful to say")
	require.NotContains(t, (*got)[0].text, "done for this month")
}

// Nobody wired it, or there is no ceiling: the message says only what it can
// stand behind.
func TestWithNothingWiredItSaysTheOldThing(t *testing.T) {
	store := withStore(t)
	ctx := context.Background()
	p := owner(t, store)
	send, got := recorder()

	a := squirrel.NewApplier(store, send, squirrel.Chat{}, nil)
	a.SetCoach(func(context.Context, int64, string, string, string) (string, []string, error) {
		return "", nil, errors.New("no coach available")
	})

	item := itemOf("!buddy everything at once")
	item.PersonID = squirrel.Ptr(p)
	require.NoError(t, a.Apply(ctx, item, squirrel.Ptr(p)))

	require.Len(t, *got, 1)
	require.NotContains(t, (*got)[0].text, "done for this month")
}
