package squirrel

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Nothing deletes a photograph, so the only thing that can stop the volume
// filling in silence is being told.
//
// It filled in silence: Keep caps one upload at 20MB and fsyncs it properly,
// and no code path anywhere removes one. For something meant to run for years
// that is monotonic growth with no retention policy and no warning — the
// volume simply fills until a write fails, which is the first anyone hears of
// it.
func TestTheVolumeSaysWhenItIsFilling(t *testing.T) {
	p, err := OpenPhotos(t.TempDir())
	require.NoError(t, err)

	said := &bytes.Buffer{}
	was := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(said, nil)))
	t.Cleanup(func() { slog.SetDefault(was) })

	// A ceiling one photograph will not reach, and then one it will.
	p.Ceiling(1 << 20)
	_, err = p.Keep(bytes.NewReader(aJPEG(2048)), "image/jpeg")
	require.NoError(t, err)
	require.NotContains(t, said.String(), "filling",
		"it warned with the volume nearly empty")

	p.Ceiling(4096)
	_, err = p.Keep(bytes.NewReader(aJPEG(2048)), "image/jpeg")
	require.NoError(t, err)
	require.Contains(t, said.String(), "the photographs are filling their volume",
		"the volume passed its ceiling and said nothing")
}

// And a ceiling nobody set is not a ceiling. Zero means no ceiling, the same
// shape the coach's monthly budget uses.
func TestNoCeilingIsASupportedChoice(t *testing.T) {
	p, err := OpenPhotos(t.TempDir())
	require.NoError(t, err)

	said := &bytes.Buffer{}
	was := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(said, nil)))
	t.Cleanup(func() { slog.SetDefault(was) })

	for i := 0; i < 3; i++ {
		_, err = p.Keep(bytes.NewReader(aJPEG(4096)), "image/jpeg")
		require.NoError(t, err)
	}
	require.NotContains(t, said.String(), "filling")
}

func TestUsedCountsOnlyWhatWasKept(t *testing.T) {
	p, err := OpenPhotos(t.TempDir())
	require.NoError(t, err)

	bytesUsed, count := p.Used()
	require.Zero(t, count)
	require.Zero(t, bytesUsed)

	_, err = p.Keep(bytes.NewReader(aJPEG(3000)), "image/jpeg")
	require.NoError(t, err)
	_, err = p.Keep(bytes.NewReader(aJPEG(1000)), "image/jpeg")
	require.NoError(t, err)

	bytesUsed, count = p.Used()
	require.Equal(t, 2, count)
	require.Equal(t, int64(4000), bytesUsed)
}

// aJPEG is n bytes that Keep will accept.
func aJPEG(n int) []byte {
	b := []byte{0xFF, 0xD8, 0xFF}
	return append(b, []byte(strings.Repeat("x", n-len(b)))...)
}
