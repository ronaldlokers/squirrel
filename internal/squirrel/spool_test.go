package squirrel_test

import (
	"os"
	"path/filepath"
	"strings"
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
