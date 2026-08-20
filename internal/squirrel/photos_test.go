package squirrel_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ronaldlokers/squirrel/internal/squirrel"
)

// No build tag: this is files on a disk and touches no database.

func TestAPhotographIsOnTheDiskWhenKeepReturns(t *testing.T) {
	dir := t.TempDir()
	photos, err := squirrel.OpenPhotos(dir)
	require.NoError(t, err)

	name, err := photos.Keep(strings.NewReader("jpegbytes"), "image/jpeg")
	require.NoError(t, err)
	require.True(t, strings.HasSuffix(name, ".jpg"))

	// Durable when it returns, so it is readable now and not after a flush.
	b, err := os.ReadFile(filepath.Join(dir, name))
	require.NoError(t, err)
	require.Equal(t, "jpegbytes", string(b))
}

// The name is random rather than derived from anything sent. A filename from a
// phone is meaningless here and a path built from it is a traversal waiting to
// be written.
func TestTheNameOwesNothingToWhatWasSent(t *testing.T) {
	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)

	first, err := photos.Keep(strings.NewReader("a"), "image/jpeg")
	require.NoError(t, err)
	second, err := photos.Keep(strings.NewReader("b"), "image/jpeg")
	require.NoError(t, err)

	require.NotEqual(t, first, second)
	require.Equal(t, first, filepath.Base(first))
}

func TestOnlyPhotographsAreKept(t *testing.T) {
	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)

	for _, kind := range []string{"application/pdf", "text/html", "image/svg+xml", ""} {
		_, err := photos.Keep(strings.NewReader("whatever"), kind)
		require.Error(t, err, kind)
	}
	for _, kind := range []string{"image/jpeg", "image/png", "image/webp", "image/heic"} {
		_, err := photos.Keep(strings.NewReader("bytes"), kind)
		require.NoError(t, err, kind)
	}
}

// Browsers append parameters to a type sometimes, and that is not a reason to
// refuse a photograph.
func TestAParameterOnTheTypeIsNotARefusal(t *testing.T) {
	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)

	_, err = photos.Keep(strings.NewReader("bytes"), "image/jpeg; charset=binary")
	require.NoError(t, err)
	require.Equal(t, "image/jpeg", squirrel.PhotoKind("IMAGE/JPEG ; x=1"))
}

// A guard rather than a rule about what you may keep — and stopping one byte
// over means a wrong Content-Length cannot fill the volume.
func TestATooBigPhotographIsRefusedAndLeavesNothingBehind(t *testing.T) {
	dir := t.TempDir()
	photos, err := squirrel.OpenPhotos(dir)
	require.NoError(t, err)

	_, err = photos.Keep(strings.NewReader(strings.Repeat("x", (8<<20)+1)), "image/jpeg")
	require.Error(t, err)

	left, err := os.ReadDir(dir)
	require.NoError(t, err)
	require.Empty(t, left, "a refused photograph left litter on the volume")
}

func TestAnEmptyPhotographIsRefused(t *testing.T) {
	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)

	_, err = photos.Keep(strings.NewReader(""), "image/jpeg")
	require.Error(t, err)
}

// The name is checked rather than trusted even though it comes from our own
// database: it is the only string in this product that becomes a path.
func TestOpenRefusesAnythingThatIsNotAPlainName(t *testing.T) {
	photos, err := squirrel.OpenPhotos(t.TempDir())
	require.NoError(t, err)

	for _, name := range []string{
		"../../etc/passwd", "/etc/passwd", "sub/photo.jpg", "", ".hidden",
	} {
		_, err := photos.Open(name)
		require.Error(t, err, name)
	}
}

// Nowhere to put them is a supported state, and it is found out at boot rather
// than at the first photograph.
func TestAnUnwritableDirectoryIsRefusedUpFront(t *testing.T) {
	_, err := squirrel.OpenPhotos("")
	require.Error(t, err)

	dir := filepath.Join(t.TempDir(), "locked")
	require.NoError(t, os.Mkdir(dir, 0o500))
	t.Cleanup(func() { _ = os.Chmod(dir, 0o700) })

	if os.Geteuid() == 0 {
		t.Skip("root writes anywhere, so there is nothing to refuse")
	}
	_, err = squirrel.OpenPhotos(dir)
	require.Error(t, err)
}
