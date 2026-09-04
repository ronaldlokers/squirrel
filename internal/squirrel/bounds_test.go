package squirrel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEveryConnectionCarriesItsBounds(t *testing.T) {
	cfg, err := poolConfigFor("postgres://squirrel:squirrel@localhost:5432/squirrel")
	require.NoError(t, err)

	require.Equal(t, "5000", cfg.ConnConfig.RuntimeParams["statement_timeout"],
		"a stuck query has to end on its own")
	require.Equal(t, 5*time.Second, cfg.ConnConfig.ConnectTimeout,
		"dialling a database that is not answering has to end on its own")
	require.Equal(t, int32(10), cfg.MaxConns,
		"a pool sized to the machine is a pool nothing bounds")
}

func TestTheAddressMayCarryItsOwnBounds(t *testing.T) {
	cfg, err := poolConfigFor(
		"postgres://squirrel:squirrel@localhost:5432/squirrel?statement_timeout=250&pool_max_conns=3&connect_timeout=2")
	require.NoError(t, err)

	require.Equal(t, "250", cfg.ConnConfig.RuntimeParams["statement_timeout"])
	require.Equal(t, int32(3), cfg.MaxConns)
	require.Equal(t, 2*time.Second, cfg.ConnConfig.ConnectTimeout)
}

func TestTheServerStopsWritingEventually(t *testing.T) {
	s := NewServer(nil)
	port, err := s.Listen("127.0.0.1:0")
	require.NoError(t, err)
	require.NotZero(t, port)
	t.Cleanup(func() { _ = s.Shutdown(t.Context()) })

	require.NotZero(t, s.http.WriteTimeout, "a response nothing bounds holds the socket")
	require.NotZero(t, s.http.IdleTimeout, "a kept-alive connection nothing bounds is the same leak")
}
