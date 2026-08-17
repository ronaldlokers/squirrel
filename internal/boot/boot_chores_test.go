//go:build integration

package boot_test

import (
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func itoa(i int) string { return strconv.Itoa(i) }

// End to end: define a chore by chat, and see it in the database. The reply
// itself goes to Campfire, which is not running here, so the assertion is on
// the state rather than on the message.
func TestBootDefinesAChoreFromChat(t *testing.T) {
	store := withStore(t)
	s := boots(t, envFor(t, nil))

	body := strings.Replace(payload, `"plain": "buy milk"`, `"plain": "every 2 weeks: vacuum"`, 1)
	res, err := http.Post(
		"http://127.0.0.1:"+itoa(s.Port())+"/transports/campfire",
		"application/json", strings.NewReader(body))
	require.NoError(t, err)
	defer res.Body.Close()

	got, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	// Since Task 1 the receipt is a boost, so the response posts nothing.
	require.Empty(t, res.Header.Get("Content-Type"))
	require.Empty(t, string(got))

	require.Eventually(t, func() bool {
		var n int
		if err := store.Pool().QueryRow(context.Background(),
			`select count(*) from chores where lower(name) = 'vacuum'`).Scan(&n); err != nil {
			return false
		}
		return n == 1
	}, 10*time.Second, 100*time.Millisecond)
}
