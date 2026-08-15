package squirrel

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// randomID names a capture whose transport gave it no id of its own. Built on
// crypto/rand rather than a uuid package, because the one-dependency rule is
// the reason this port exists.
func randomID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand does not fail in practice, and a monotonic fallback is
		// still better than a panic on the capture path.
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}

// ErrMalformedSpoolFile marks a file that will never parse. The drain treats
// it as permanent and quarantines rather than retrying it forever.
var ErrMalformedSpoolFile = errors.New("malformed spool file")

const quarantineDir = "quarantine"

type Spool struct {
	dir string
}

func OpenSpool(dir string) (*Spool, error) {
	if err := os.MkdirAll(filepath.Join(dir, quarantineDir), 0o750); err != nil {
		return nil, fmt.Errorf("opening spool: %w", err)
	}
	return &Spool{dir: dir}, nil
}

var unsafeChars = regexp.MustCompile(`[^A-Za-z0-9_-]`)

// safe sanitises for the filename only. The real id is inside the file.
func safe(v string) string {
	v = unsafeChars.ReplaceAllString(v, "_")
	if len(v) > 64 {
		v = v[:64]
	}
	return v
}

func (s *Spool) filename(c Capture) string {
	id := randomID()
	if c.ExternalID != nil {
		id = safe(*c.ExternalID)
	}
	// Zero-padded epoch millis, so lexical order is chronological order.
	return fmt.Sprintf("%015d-%s-%s.json", c.ReceivedAt.UnixMilli(), safe(c.Transport), id)
}

// Write is durable when it returns.
//
// Write, fsync the file, rename, then fsync the directory. The rename is
// atomic so the drain sees either nothing or a whole file. The directory sync
// is what makes the rename survive a host power loss rather than only a
// process crash — this runs on Raspberry Pis without a UPS.
//
// The temp file is exclusive to this call (os.CreateTemp, not a deterministic
// "<name>.tmp" path): name is derived from safe(externalID), and safe is not
// injective — two different external ids (e.g. two Matrix ids differing only
// past the 64-byte truncation, or only in a character safe() maps to "_")
// can produce the same name. net/http serves each request in its own
// goroutine, so two such writes genuinely race. A shared, non-exclusive temp
// path let concurrent writers truncate and overwrite each other's bytes on
// the same inode, so both Write calls could return success while the surviving
// file was a torn mix of neither payload. A unique temp file per call means
// each writer's own bytes are only ever visible to itself until the final
// rename, which is atomic — so once names collide, last-writer-wins is a
// clean, whole-file overwrite rather than a corrupted one.
func (s *Spool) Write(c Capture) (string, error) {
	name := s.filename(c)

	encoded, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("encoding capture: %w", err)
	}

	f, err := os.CreateTemp(s.dir, name+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("creating spool file: %w", err)
	}
	temporary := f.Name()

	if _, err = f.Write(encoded); err == nil {
		err = f.Sync()
	}
	if closeErr := f.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(temporary)
		return "", fmt.Errorf("writing spool file: %w", err)
	}

	if err := os.Rename(temporary, filepath.Join(s.dir, name)); err != nil {
		os.Remove(temporary)
		return "", fmt.Errorf("renaming spool file: %w", err)
	}
	if err := syncDir(s.dir); err != nil {
		return "", fmt.Errorf("syncing spool directory: %w", err)
	}
	return name, nil
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	err = d.Sync()
	if closeErr := d.Close(); err == nil {
		err = closeErr
	}
	return err
}

func (s *Spool) List() ([]string, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, fmt.Errorf("listing spool: %w", err)
	}
	names := []string{}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names, nil
}

func (s *Spool) Read(name string) (Capture, error) {
	raw, err := os.ReadFile(filepath.Join(s.dir, name))
	if err != nil {
		return Capture{}, fmt.Errorf("reading spool file: %w", err)
	}
	var c Capture
	if err := json.Unmarshal(raw, &c); err != nil {
		return Capture{}, fmt.Errorf("%w: %s: %v", ErrMalformedSpoolFile, name, err)
	}
	return c, nil
}

func (s *Spool) Remove(name string) error {
	return os.Remove(filepath.Join(s.dir, name))
}

// Quarantine moves rather than deletes. A file that cannot be inserted must
// not spin forever, and it must not disappear either.
func (s *Spool) Quarantine(name string) error {
	if err := os.Rename(filepath.Join(s.dir, name), filepath.Join(s.dir, quarantineDir, name)); err != nil {
		return err
	}
	return syncDir(filepath.Join(s.dir, quarantineDir))
}

// Sweep deletes .tmp files left by a crash and reports how many.
func (s *Spool) Sweep() (int, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return 0, err
	}
	swept := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".tmp") {
			continue
		}
		if err := os.Remove(filepath.Join(s.dir, e.Name())); err != nil {
			return swept, err
		}
		swept++
	}
	return swept, nil
}

func (s *Spool) Writable() bool {
	probe := filepath.Join(s.dir, ".writable")
	f, err := os.OpenFile(probe, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return false
	}
	f.Close()
	os.Remove(probe)
	return true
}
