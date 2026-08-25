package squirrel

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// A photograph, drawn rather than fetched: bands of colour so a scaler that
// dropped pixels instead of averaging them would be visible in the result.
func aPhotograph(w, h int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 90, A: 255})
		}
	}
	return img
}

func keptPhoto(t *testing.T, p *Photos, img image.Image) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, jpeg.Encode(&buf, img, nil))
	name, err := p.Keep(&buf, "image/jpeg")
	require.NoError(t, err)
	return name
}

func photoDir(t *testing.T) *Photos {
	t.Helper()
	p, err := OpenPhotos(t.TempDir())
	require.NoError(t, err)
	return p
}

// The long edge is capped and the shape is kept. A thumbnail that squares off
// a portrait photograph is a different picture.
func TestShrinkCapsTheLongEdgeAndKeepsTheShape(t *testing.T) {
	wide := shrink(aPhotograph(2000, 1000), 640).Bounds()
	require.Equal(t, 640, wide.Dx())
	require.Equal(t, 320, wide.Dy(), "the shape changed")

	tall := shrink(aPhotograph(1000, 2000), 640).Bounds()
	require.Equal(t, 640, tall.Dy())
	require.Equal(t, 320, tall.Dx(), "the shape changed")
}

// Already small enough is left alone, rather than re-encoded into something
// softer than it arrived.
func TestShrinkLeavesASmallPictureAlone(t *testing.T) {
	src := aPhotograph(300, 200)
	require.Same(t, src, shrink(src, 640))
}

// Averaged, not sampled. A scaler that picked one source pixel per
// destination pixel would give a corner the colour of a single original
// pixel; averaging a block of a gradient gives the block's mean.
func TestShrinkAveragesRatherThanPicks(t *testing.T) {
	// Ten source pixels per destination pixel across: x runs 0..9 in the
	// first block, so the mean is 4.5 and a picker would give 0 or 9.
	got := shrink(aPhotograph(6400, 10), 640)
	r, _, _, _ := got.At(0, 0).RGBA()
	require.InDelta(t, 4.5, float64(r>>8), 1.0, "it sampled instead of averaging")
}

// The copy lands beside the original, and it is very much smaller.
func TestAThumbnailIsMadeBesideTheOriginalAndIsSmaller(t *testing.T) {
	p := photoDir(t)
	name := keptPhoto(t, p, aPhotograph(2400, 1800))

	f, err := p.Thumb(name)
	require.NoError(t, err)
	defer f.Close()

	small, err := f.Stat()
	require.NoError(t, err)
	big, err := os.Stat(filepath.Join(p.dir, name))
	require.NoError(t, err)
	require.Less(t, small.Size(), big.Size()/2, "the copy saved nothing worth serving")

	// And it is a picture, at the capped size.
	img, _, err := image.Decode(f)
	require.NoError(t, err)
	require.Equal(t, 640, img.Bounds().Dx())
}

// The original is never touched. Capture is sacred; the thumbnail is derived.
func TestMakingAThumbnailLeavesTheOriginalAlone(t *testing.T) {
	p := photoDir(t)
	name := keptPhoto(t, p, aPhotograph(1200, 900))
	before, err := os.ReadFile(filepath.Join(p.dir, name))
	require.NoError(t, err)

	_, err = p.Thumb(name)
	require.NoError(t, err)

	after, err := os.ReadFile(filepath.Join(p.dir, name))
	require.NoError(t, err)
	require.Equal(t, before, after)
}

// Asked twice, made once. The second call opens the file that is already
// there rather than re-encoding the original.
func TestAThumbnailIsMadeOnceAndThenOpened(t *testing.T) {
	p := photoDir(t)
	name := keptPhoto(t, p, aPhotograph(1200, 900))

	first, err := p.Thumb(name)
	require.NoError(t, err)
	first.Close()
	made, err := os.Stat(filepath.Join(p.dir, ThumbName(name)))
	require.NoError(t, err)

	// Removing the original proves the second call did not read it.
	require.NoError(t, os.Remove(filepath.Join(p.dir, name)))
	second, err := p.Thumb(name)
	require.NoError(t, err)
	defer second.Close()

	again, err := second.Stat()
	require.NoError(t, err)
	require.Equal(t, made.Size(), again.Size())
}

// A png is a photograph too — a screenshot, most often.
func TestAPngGetsAThumbnailAsWell(t *testing.T) {
	p := photoDir(t)
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, aPhotograph(1400, 1400)))
	name, err := p.Keep(&buf, "image/png")
	require.NoError(t, err)

	f, err := p.Thumb(name)
	require.NoError(t, err)
	defer f.Close()
	img, _, err := image.Decode(f)
	require.NoError(t, err)
	require.Equal(t, 640, img.Bounds().Dx())
}

// A kind Go cannot decode says so, so the caller can serve the original
// rather than a broken image.
func TestAKindThatCannotBeDecodedSaysSo(t *testing.T) {
	p := photoDir(t)
	name, err := p.Keep(bytes.NewReader([]byte("ftypheic not really")), "image/heic")
	require.NoError(t, err)

	_, err = p.Thumb(name)
	require.Error(t, err)
	require.NoFileExists(t, filepath.Join(p.dir, ThumbName(name)),
		"a half-made thumbnail was left behind")
}

// The same name check Open carries. This one becomes a path too, and it is
// reached by a URL — so the test is a name that would otherwise resolve, not
// one that fails on its own. A file that really is there, one directory up.
func TestAThumbnailNameIsNotAPath(t *testing.T) {
	p := photoDir(t)
	// A real photograph, so the guard is the only thing stopping it. Bytes
	// that cannot be decoded would fail for the wrong reason and the test
	// would pass with the check deleted.
	var real bytes.Buffer
	require.NoError(t, jpeg.Encode(&real, aPhotograph(80, 60), nil))
	up := filepath.Dir(p.dir)
	require.NoError(t, os.WriteFile(filepath.Join(up, "secret.jpg"), real.Bytes(), 0o600))

	_, err := p.Thumb("../secret.jpg")
	require.Error(t, err, "it reached outside its own directory")

	// And a hidden file inside it, which is where the temporaries live.
	require.NoError(t, os.WriteFile(filepath.Join(p.dir, ".writable-x"), real.Bytes(), 0o600))
	_, err = p.Thumb(".writable-x")
	require.Error(t, err)

	_, err = p.Thumb("")
	require.Error(t, err)
}
