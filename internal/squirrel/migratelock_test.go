//go:build integration

package squirrel

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"testing/fstest"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConcurrentMigrationPassesAreSerialized(t *testing.T) {
	ctx := context.Background()
	store := storeForMigrations(t)

	stamp := time.Now().UnixNano()
	nameA := fmt.Sprintf("migrations/%d_a.sql", stamp)
	nameB := fmt.Sprintf("migrations/%d_b.sql", stamp)
	t.Cleanup(func() {
		_, _ = store.pool.Exec(context.Background(),
			`delete from schema_migrations where version in ($1, $2)`, nameA, nameB)
	})
	one := fstest.MapFS{nameA: {Data: []byte("select pg_sleep(0.3);")}}
	two := fstest.MapFS{nameB: {Data: []byte("select pg_sleep(0.3);")}}

	var wg sync.WaitGroup
	wg.Add(2)
	start := time.Now()
	go func() {
		defer wg.Done()
		require.NoError(t, store.migrateFrom(ctx, one))
	}()
	go func() {
		defer wg.Done()
		require.NoError(t, store.migrateFrom(ctx, two))
	}()
	wg.Wait()
	elapsed := time.Since(start)

	require.GreaterOrEqual(t, elapsed, 550*time.Millisecond,
		"two migration passes ran concurrently rather than being serialized by a lock")

	var recorded int
	require.NoError(t, store.pool.QueryRow(ctx,
		`select count(*) from schema_migrations where version in ($1, $2)`,
		nameA, nameB).Scan(&recorded))
	require.Equal(t, 2, recorded)
}
