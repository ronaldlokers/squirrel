package squirrel

import (
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
)

// A smaller copy, because the card is 260 pixels wide and the original is a
// photograph from a phone.
//
// The note carries the photograph at card size everywhere it is drawn, and
// serving eight megabytes into a 260-pixel box is the whole of the cost with
// none of the benefit. This is the same picture at a size the card can use.
//
// Nothing here ever replaces the original. A photograph attached to a note is
// part of what was captured and capture is sacred — the thumbnail is a derived
// file that can be deleted at any time and will simply be made again.

// thumbWide is the long edge of the copy, in pixels.
//
// 640 rather than 260: the card is 260 CSS pixels and a phone draws it at two
// or three device pixels each, so a 260-pixel file is the one thing worse than
// a large one — visibly soft on exactly the screen this product is used on.
const thumbWide = 640

// thumbQuality is JPEG quality for the copy. 80 is where a downscaled
// photograph stops getting visibly better and the file keeps growing.
const thumbQuality = 80

// ThumbName is what the smaller copy is called on disk, beside the original.
//
// Derived from the original's name rather than random, so a thumbnail can
// always be found from the row without storing a second column — and so a
// directory listing shows which original an orphan belongs to.
func ThumbName(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name)) + ".thumb.jpg"
}

// Thumb opens the smaller copy, making it first if it is not there yet.
//
// Made on demand rather than at capture, for two reasons worth stating. Every
// photograph taken before this existed has none, and a migration that walks a
// volume re-encoding images is a great deal of machinery for a file that can
// be rebuilt from its original in a few milliseconds. And a capture must never
// fail because a decoder did not like something — the bytes are kept first and
// asked about later.
//
// A kind Go cannot decode has no thumbnail and never will. HEIC and WebP are
// both in the accepted list and neither is in the standard library; rather
// than take a dependency to shrink a file, the caller is told there is none
// and serves the original. That is a larger download for those two kinds and
// it is the honest answer.
func (p *Photos) Thumb(name string) (*os.File, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return nil, fmt.Errorf("not a photo name: %q", name)
	}
	thumb := ThumbName(name)
	if f, err := os.Open(filepath.Join(p.dir, thumb)); err == nil {
		return f, nil
	}
	if err := p.makeThumb(name, thumb); err != nil {
		return nil, err
	}
	return os.Open(filepath.Join(p.dir, thumb))
}

// makeThumb decodes the original, shrinks it and writes the copy the same way
// Keep writes an original: a temporary file, fsynced, renamed into place. A
// half-written thumbnail served to a browser is a broken image on a card, and
// the rename is what makes that impossible.
func (p *Photos) makeThumb(name, thumb string) error {
	in, err := os.Open(filepath.Join(p.dir, name))
	if err != nil {
		return err
	}
	defer in.Close()

	src, _, err := image.Decode(in)
	if err != nil {
		return fmt.Errorf("that photograph cannot be shrunk: %w", err)
	}

	out, err := os.CreateTemp(p.dir, "thumb-*.tmp")
	if err != nil {
		return err
	}
	temporary := out.Name()
	defer func() { _ = os.Remove(temporary) }()

	if err := jpeg.Encode(out, shrink(src, thumbWide), &jpeg.Options{Quality: thumbQuality}); err != nil {
		_ = out.Close()
		return fmt.Errorf("writing a thumbnail: %w", err)
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, filepath.Join(p.dir, thumb)); err != nil {
		return err
	}
	return syncDir(p.dir)
}

// shrink scales an image down so its long edge is at most wide, by averaging
// each destination pixel over the source pixels it covers.
//
// Hand-rolled rather than a dependency. The standard library draws and encodes
// but does not resample, and the one filter that matters here is the one that
// is correct for shrinking: averaging the covered area, which is what a box
// filter is. Taking a module for a page of arithmetic that only ever runs
// downwards is not a trade this needs to make.
//
// An image already small enough comes back untouched, so a screenshot pasted
// in at 300 pixels is not re-encoded into something softer than it started.
func shrink(src image.Image, wide int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= wide && h <= wide {
		return src
	}
	dw, dh := w, h
	if w >= h {
		dw, dh = wide, max(1, h*wide/w)
	} else {
		dw, dh = max(1, w*wide/h), wide
	}

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := range dh {
		// The source rows this destination row covers. Half-open, and at
		// least one pixel tall — a destination taller than one pixel per
		// source row cannot happen here, but a zero-height average would be a
		// divide by zero and this is cheaper than proving it never occurs.
		y0, y1 := y*h/dh, max(y*h/dh+1, (y+1)*h/dh)
		for x := range dw {
			x0, x1 := x*w/dw, max(x*w/dw+1, (x+1)*w/dw)
			var r, g, bl, a uint64
			var n uint64
			for sy := y0; sy < y1; sy++ {
				for sx := x0; sx < x1; sx++ {
					sr, sg, sb, sa := src.At(b.Min.X+sx, b.Min.Y+sy).RGBA()
					r, g, bl, a, n = r+uint64(sr), g+uint64(sg), bl+uint64(sb), a+uint64(sa), n+1
				}
			}
			// RGBA() returns 16-bit values; the destination takes 8.
			dst.Set(x, y, color.RGBA{
				R: uint8(r / n >> 8), G: uint8(g / n >> 8),
				B: uint8(bl / n >> 8), A: uint8(a / n >> 8),
			})
		}
	}
	return dst
}
