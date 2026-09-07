//go:build integration

package squirrel

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type queryCountingTracer struct {
	match func(string) bool
	count atomic.Int64
}

func (c *queryCountingTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	if c.match(data.SQL) {
		c.count.Add(1)
	}
	return ctx
}

func (c *queryCountingTracer) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func TestMigrateChecksAppliedVersionsInOneRoundTripNotOnePerMigration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	require.NotEmpty(t, url, "TEST_DATABASE_URL is required — see docs/testing.md")

	cfg, err := poolConfigFor(url)
	require.NoError(t, err)

	tracer := &queryCountingTracer{match: func(sql string) bool {
		return strings.Contains(sql, "from schema_migrations")
	}}
	cfg.ConnConfig.Tracer = tracer

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	store := &Store{pool: pool}

	stamp := fmt.Sprintf("migrations/%d", 9200)
	files := fstest.MapFS{}
	for i := range 40 {
		files[fmt.Sprintf("%s_%02d.sql", stamp, i)] = &fstest.MapFile{Data: []byte("select 1;")}
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(),
			`delete from schema_migrations where version like $1`, stamp+"%")
	})

	require.NoError(t, store.migrateFrom(context.Background(), files))

	require.Equal(t, int64(1), tracer.count.Load(),
		"checking which of 40 migrations are already applied made more than one round trip to the database")
}
