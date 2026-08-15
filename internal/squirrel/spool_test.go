package squirrel_test

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

func openSpool(t *testing.T) (*squirrel.Spool, string) {
	t.Helper()
	dir := t.TempDir()
	sp, err := squirrel.OpenSpool(dir)
	require.NoError(t, err)
	return sp, dir
}

func TestSpoolRoundTrip(t *testing.T) {
	sp, _ := openSpool(t)

	name, err := sp.Write(capture(nil))
	require.NoError(t, err)

	names, err := sp.List()
	require.NoError(t, err)
	require.Equal(t, []string{name}, names)

	got, err := sp.Read(name)
	require.NoError(t, err)
	require.Equal(t, capture(nil), got)
}

func TestSpoolSortsChronologically(t *testing.T) {
	sp, _ := openSpool(t)

	later, err := sp.Write(capture(func(c *squirrel.Capture) {
		c.ExternalID = squirrel.Ptr("43")
		c.ReceivedAt = c.ReceivedAt.Add(time.Hour)
	}))
	require.NoError(t, err)
	earlier, err := sp.Write(capture(nil))
	require.NoError(t, err)

	names, err := sp.List()
	require.NoError(t, err)
	require.Equal(t, []string{earlier, later}, names)
}

func TestSpoolLeavesNoTempFile(t *testing.T) {
	sp, dir := openSpool(t)
	_, err := sp.Write(capture(nil))
	require.NoError(t, err)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.False(t, strings.HasSuffix(e.Name(), ".tmp"), e.Name())
	}
}

// A partial file must be invisible to the drain.
func TestSpoolNeverListsATempFile(t *testing.T) {
	sp, dir := openSpool(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "000000000000001-campfire-9.json.tmp"), []byte("{ partial"), 0o600))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

func TestSpoolSweepsTempFiles(t *testing.T) {
	sp, dir := openSpool(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "000000000000001-campfire-9.json.tmp"), []byte("{ partial"), 0o600))

	swept, err := sp.Sweep()
	require.NoError(t, err)
	require.Equal(t, 1, swept)

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		require.False(t, strings.HasSuffix(e.Name(), ".tmp"), e.Name())
	}
}

// Matrix room ids carry ! and :, Telegram chat ids are negative. None of that
// may reach a filename; the real id lives inside the file.
func TestSpoolSanitisesTheExternalID(t *testing.T) {
	sp, _ := openSpool(t)

	name, err := sp.Write(capture(func(c *squirrel.Capture) {
		c.Transport = "matrix"
		c.ExternalID = squirrel.Ptr("!a:b/../c")
	}))
	require.NoError(t, err)
	require.NotContains(t, name, "/")
	require.NotContains(t, name, ":")

	got, err := sp.Read(name)
	require.NoError(t, err)
	require.Equal(t, "!a:b/../c", *got.ExternalID)
}

func TestSpoolGivesUnknownIDsUniqueNames(t *testing.T) {
	sp, _ := openSpool(t)

	first, err := sp.Write(capture(func(c *squirrel.Capture) { c.ExternalID = nil }))
	require.NoError(t, err)
	second, err := sp.Write(capture(func(c *squirrel.Capture) { c.ExternalID = nil }))
	require.NoError(t, err)
	require.NotEqual(t, first, second)

	names, err := sp.List()
	require.NoError(t, err)
	require.Len(t, names, 2)
}

// The reviewer's reproduction: safe() is not injective, so two different
// external ids — here "!a:b" and "!a/b", two Matrix room ids that both
// sanitise to "_a_b" — can drive Write to the exact same filename. net/http
// serves each request in its own goroutine, so concurrent writers really can
// race on that one name. Before the fix, the temp path was a deterministic
// "<name>.tmp" shared by every racer; two writers truncating and writing the
// same inode at once could leave a torn, corrupted file behind while both
// Write calls still reported success. This launches 16 such writers at once
// and checks that never happens: every Write must succeed, no partial temp
// file may be left over, and the single file left at the shared name must be
// a whole, untampered copy of exactly one writer's capture — never a mix.
//
// A forced full collision means only one writer's content can be on disk
// once every goroutine has finished (the final rename is last-writer-wins by
// design, see spool.go). So this cannot assert that every individual writer's
// own capture survives — only that whichever one does is intact.
func TestSpoolConcurrentWritesDoNotCollideOnAFilename(t *testing.T) {
	sp, dir := openSpool(t)

	const n = 16
	receivedAt := time.Date(2026, 8, 14, 9, 31, 4, 0, time.UTC)
	variants := []string{"!a:b", "!a/b"}

	captures := make([]squirrel.Capture, n)
	names := make([]string, n)
	errs := make([]error, n)

	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		captures[i] = capture(func(c *squirrel.Capture) {
			c.Transport = "matrix"
			c.ReceivedAt = receivedAt
			c.ExternalID = squirrel.Ptr(variants[i%len(variants)])
			c.Text = fmt.Sprintf("capture-%d", i)
		})
	}

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			names[i], errs[i] = sp.Write(captures[i])
		}(i)
	}
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i], "writer %d", i)
		require.Equal(t, names[0], names[i], "writer %d computed a different name", i)
	}

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var jsonFiles int
	for _, e := range entries {
		require.False(t, strings.HasSuffix(e.Name(), ".tmp"), "stray temp file: %s", e.Name())
		if strings.HasSuffix(e.Name(), ".json") {
			jsonFiles++
		}
	}
	require.Equal(t, 1, jsonFiles, "exactly one file should survive the collision")

	got, err := sp.Read(names[0])
	require.NoError(t, err, "surviving file must not be corrupted")

	matched := false
	for i := 0; i < n; i++ {
		if reflect.DeepEqual(got, captures[i]) {
			matched = true
			break
		}
	}
	require.True(t, matched, "surviving capture %+v must exactly equal one writer's capture, not a mix", got)
}

func TestSpoolRemoves(t *testing.T) {
	sp, _ := openSpool(t)
	name, err := sp.Write(capture(nil))
	require.NoError(t, err)
	require.NoError(t, sp.Remove(name))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)
}

// Quarantine moves rather than deletes: nothing is ever thrown away.
func TestSpoolQuarantines(t *testing.T) {
	sp, dir := openSpool(t)
	name, err := sp.Write(capture(nil))
	require.NoError(t, err)
	require.NoError(t, sp.Quarantine(name))

	names, err := sp.List()
	require.NoError(t, err)
	require.Empty(t, names)

	entries, err := os.ReadDir(filepath.Join(dir, "quarantine"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, name, entries[0].Name())
}

func TestSpoolReportsMalformedFiles(t *testing.T) {
	sp, dir := openSpool(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "000000000000001-campfire-9.json"), []byte("{ not json"), 0o600))

	_, err := sp.Read("000000000000001-campfire-9.json")
	require.ErrorIs(t, err, squirrel.ErrMalformedSpoolFile)
}

func TestSpoolReportsWritable(t *testing.T) {
	sp, _ := openSpool(t)
	require.True(t, sp.Writable())
}
