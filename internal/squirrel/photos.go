package squirrel

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Where a photograph lives.
//
// On a volume beside the pod rather than in object storage, decided on 20
// August 2026: this cluster has no object storage and adding some would mean a
// service to patch, back up and keep alive for one feature. What it costs is
// stated rather than discovered — photographs join the pod's lifecycle, and
// the restore drill has to grow to cover the volume.
//
// The bytes are written before anything references them, and fsynced, for the
// same reason the spool is: a capture that says it was kept and was not is the
// one failure this product does not have.

// Photos is a directory of them. The zero value is unusable; New returns one
// that has checked it can write.
type Photos struct {
	dir string
	// ceiling is where "this volume is filling" starts, in bytes. Zero is no
	// ceiling and the default.
	ceiling int64
}

// biggest is the ceiling on one photograph, and a guard rather than a rule
// about what you may keep.
//
// Twenty megabytes. Eight was the first number here and it was chosen as "a
// generous phone photo" from memory rather than from a phone: a current camera
// at full resolution clears it without trying, and the whole case for this
// feature is photographing something rather than typing it out. A ceiling that
// the ordinary case trips over is not a guard, it is a defect.
//
// It costs nothing to be generous now that the bytes stream from the socket
// onto the volume — there is no copy of this anywhere else on the way past.
const biggest = 20 << 20

// kinds are what may be stored, by what the browser says it is sending.
//
// A closed list rather than a check on the bytes: this is one person's own
// camera, and the risk being managed is a mistake rather than an attack. What
// the list actually buys is that the serving side can answer with a type it
// chose from here, so nothing the browser said is ever echoed back as a
// content type.
var kinds = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
	"image/heic": ".heic",
	"image/heif": ".heif",
}

// KnownKind reports whether this is something we will keep, and what it is
// called on disk.
func KnownKind(contentType string) (string, bool) {
	// Browsers send "image/jpeg" but a few append parameters.
	base, _, _ := strings.Cut(contentType, ";")
	ext, ok := kinds[strings.ToLower(strings.TrimSpace(base))]
	return ext, ok
}

// PhotoKind is the stored type for what a browser claimed, normalised to the
// list this keeps. What goes in the row is chosen here rather than copied from
// the request, so the serving side can hand it straight back.
func PhotoKind(contentType string) string {
	base, _, _ := strings.Cut(contentType, ";")
	base = strings.ToLower(strings.TrimSpace(base))
	if _, ok := kinds[base]; ok {
		return base
	}
	return ""
}

// OpenPhotos prepares the directory, and refuses if it cannot be written to.
//
// Refusing here rather than at the first photograph, for the reason the spool
// refuses at mount: the first photograph is the worst possible moment to find
// out that there is nowhere to put it.
func OpenPhotos(dir string) (*Photos, error) {
	if dir == "" {
		return nil, errors.New("no photo directory configured")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("preparing the photo directory: %w", err)
	}
	p := &Photos{dir: dir}
	if !p.Writable() {
		return nil, fmt.Errorf("photo directory is not writable: %s", dir)
	}
	return p, nil
}

// Writable reports whether a photograph could be stored right now.
func (p *Photos) Writable() bool {
	f, err := os.CreateTemp(p.dir, ".writable-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// Used is what the store is holding: total bytes and how many photographs.
//
// A flat directory read rather than a walk, because it is flat, and cheap
// enough to do on the path that makes it grow. Errors are swallowed on
// purpose — this exists to warn, and a warning that can fail a capture would
// be worse than the thing it warns about.
func (p *Photos) Used() (bytes int64, count int) {
	entries, err := os.ReadDir(p.dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		bytes += info.Size()
		count++
	}
	return bytes, count
}

// Ceiling is the point at which this starts saying the volume is filling, in
// bytes. Zero means no ceiling, which is a supported choice and the default —
// the same shape the coach's monthly budget uses.
//
// Nothing here ever deletes a photograph. A photograph attached to a note is
// part of what was captured, and capture is sacred; what was missing was not
// pruning but a signal, because the volume simply filled in silence until a
// write failed. This is that signal, and it is for the person who runs the
// thing rather than the person using it.
func (p *Photos) Ceiling(bytes int64) { p.ceiling = bytes }

// warnIfFilling says so once the store passes the configured share of its
// ceiling. Called after a successful write, which is the only moment the
// answer can have changed.
func (p *Photos) warnIfFilling() {
	if p.ceiling <= 0 {
		return
	}
	used, count := p.Used()
	if used*10 < p.ceiling*8 {
		return
	}
	slog.Warn("the photographs are filling their volume",
		"used", used, "ceiling", p.ceiling, "photographs", count,
		"note", "nothing is deleted; this is the point to grow the volume")
}

// Keep stores one photograph and hands back the name to remember it by.
//
// Durable when it returns: written, fsynced, then renamed into place. The same
// shape the spool uses, and for the same reason — these run on Raspberry Pis
// with no UPS, and a rename is the only way to be sure a reader sees either
// nothing or a whole file.
//
// The name is random rather than derived from anything you sent. A filename
// from a phone is attacker-controlled in the general case and meaningless in
// this one, and a path built from it is a path traversal waiting to be
// written; a random name is neither.
func (p *Photos) Keep(r io.Reader, contentType string) (string, error) {
	ext, ok := KnownKind(contentType)
	if !ok {
		return "", fmt.Errorf("not a photograph this keeps: %q", contentType)
	}

	f, err := os.CreateTemp(p.dir, "photo-*.tmp")
	if err != nil {
		return "", fmt.Errorf("making room for a photo: %w", err)
	}
	temporary := f.Name()
	defer func() { _ = os.Remove(temporary) }()

	// One byte over the ceiling is enough to know it is over, and stopping
	// there means a wrong Content-Length cannot fill the volume.
	written, err := io.Copy(f, io.LimitReader(r, biggest+1))
	if err != nil {
		_ = f.Close()
		return "", fmt.Errorf("writing a photo: %w", err)
	}
	if written > biggest {
		_ = f.Close()
		return "", fmt.Errorf("that photo is too big")
	}
	if written == 0 {
		_ = f.Close()
		return "", errors.New("that photo is empty")
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return "", fmt.Errorf("flushing a photo: %w", err)
	}
	if err := f.Close(); err != nil {
		return "", fmt.Errorf("closing a photo: %w", err)
	}

	name := randomName() + ext
	if err := os.Rename(temporary, filepath.Join(p.dir, name)); err != nil {
		return "", fmt.Errorf("keeping a photo: %w", err)
	}
	if err := syncDir(p.dir); err != nil {
		return "", err
	}
	// After the write, because that is the only moment the answer can have
	// changed, and after it has succeeded, because a warning must never be
	// what stops a photograph being kept.
	p.warnIfFilling()
	return name, nil
}

// Open reads one back, by the name Keep returned.
//
// The name is checked rather than trusted even though it comes from our own
// database: it is the only string in this product that becomes a path, and a
// row is one bad migration away from holding something else.
func (p *Photos) Open(name string) (*os.File, error) {
	if name == "" || name != filepath.Base(name) || strings.HasPrefix(name, ".") {
		return nil, fmt.Errorf("not a photo name: %q", name)
	}
	return os.Open(filepath.Join(p.dir, name))
}

func randomName() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand does not fail on the platforms this runs on, and a
		// photograph is not worth a panic. A timestamp is unique enough to
		// avoid a collision in the one case this could ever happen.
		return fmt.Sprintf("photo-%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

// PhotoModTime is when the file was written, for conditional requests. A
// missing stat is not worth failing a photograph over — the zero time simply
// means the browser revalidates.
func PhotoModTime(f *os.File) time.Time {
	info, err := f.Stat()
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}
