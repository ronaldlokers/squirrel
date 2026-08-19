//go:build browser

package web

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// A DevTools client, in about a hundred lines, because the alternatives were
// worse.
//
// --dump-dom looked like it would do: no protocol, no dependency. It does not.
// It serialises the page before deferred scripts have had their effect, and
// with a service worker registered the virtual clock never expires, so the
// browser has to be killed. Both failures look like "the script did nothing",
// which is exactly the answer a test must never give wrongly.
//
// A library would be one import and several hundred dependencies for a test
// binary. The protocol itself is a websocket carrying JSON, and this file is
// the subset of it these tests use: connect, evaluate, dispatch a key, wait.
type cdp struct {
	conn net.Conn
	rw   *bufio.ReadWriter
	next int
}

// dialCDP opens the page target's websocket. The browser is told which port to
// listen on, so this is a plain HTTP request to it and an upgrade of the
// answer's URL.
func dialCDP(t *testing.T, port int, said fmt.Stringer) *cdp {
	t.Helper()

	var targets []struct {
		Type string `json:"type"`
		WS   string `json:"webSocketDebuggerUrl"`
	}
	deadline := time.Now().Add(30 * time.Second)
	for {
		res, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/json", port))
		if err == nil {
			err = json.NewDecoder(res.Body).Decode(&targets)
			res.Body.Close()
		}
		if err == nil && len(targets) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no devtools target after 30s: %v\nthe browser said:\n%s", err, said)
		}
		time.Sleep(100 * time.Millisecond)
	}

	var ws string
	for _, target := range targets {
		if target.Type == "page" && target.WS != "" {
			ws = target.WS
		}
	}
	if ws == "" {
		t.Fatal("the browser has no page target")
	}

	u, err := url.Parse(ws)
	if err != nil {
		t.Fatal(err)
	}
	conn, err := net.Dial("tcp", u.Host)
	if err != nil {
		t.Fatal(err)
	}

	key := make([]byte, 16)
	_, _ = rand.Read(key)
	handshake := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\n"+
		"Connection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		u.RequestURI(), u.Host, base64.StdEncoding.EncodeToString(key))
	if _, err := io.WriteString(conn, handshake); err != nil {
		t.Fatal(err)
	}

	rw := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	res, err := http.ReadResponse(rw.Reader, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusSwitchingProtocols {
		t.Fatalf("the browser refused the upgrade: %s", res.Status)
	}

	c := &cdp{conn: conn, rw: rw}
	t.Cleanup(func() { conn.Close() })
	return c
}

// write sends one masked text frame. Every client frame must be masked; that
// is the whole of what this direction needs.
func (c *cdp) write(t *testing.T, payload []byte) {
	t.Helper()

	header := []byte{0x81} // FIN, text
	n := len(payload)
	switch {
	case n < 126:
		header = append(header, byte(0x80|n))
	case n < 1<<16:
		header = append(header, 0x80|126, 0, 0)
		binary.BigEndian.PutUint16(header[2:], uint16(n))
	default:
		header = append(header, 0x80|127)
		header = append(header, make([]byte, 8)...)
		binary.BigEndian.PutUint64(header[2:], uint64(n))
	}

	mask := make([]byte, 4)
	_, _ = rand.Read(mask)
	header = append(header, mask...)

	masked := make([]byte, n)
	for i := range payload {
		masked[i] = payload[i] ^ mask[i%4]
	}

	if _, err := c.rw.Write(header); err != nil {
		t.Fatal(err)
	}
	if _, err := c.rw.Write(masked); err != nil {
		t.Fatal(err)
	}
	if err := c.rw.Flush(); err != nil {
		t.Fatal(err)
	}
}

// read returns the next text frame's payload. Server frames are never masked,
// and control frames are skipped rather than answered: nothing here lives long
// enough to be pinged.
func (c *cdp) read(t *testing.T) []byte {
	t.Helper()
	_ = c.conn.SetReadDeadline(time.Now().Add(30 * time.Second))

	for {
		var head [2]byte
		if _, err := io.ReadFull(c.rw, head[:]); err != nil {
			t.Fatalf("reading from the browser: %v", err)
		}
		opcode := head[0] & 0x0f
		length := int(head[1] & 0x7f)

		switch length {
		case 126:
			var ext [2]byte
			if _, err := io.ReadFull(c.rw, ext[:]); err != nil {
				t.Fatal(err)
			}
			length = int(binary.BigEndian.Uint16(ext[:]))
		case 127:
			var ext [8]byte
			if _, err := io.ReadFull(c.rw, ext[:]); err != nil {
				t.Fatal(err)
			}
			length = int(binary.BigEndian.Uint64(ext[:]))
		}

		payload := make([]byte, length)
		if _, err := io.ReadFull(c.rw, payload); err != nil {
			t.Fatal(err)
		}
		if opcode == 0x1 {
			return payload
		}
		if opcode == 0x8 {
			t.Fatal("the browser closed the connection")
		}
	}
}

// send makes one call and waits for the answer to that call, dropping the
// events that arrive in between.
func (c *cdp) send(t *testing.T, method string, params map[string]any) map[string]any {
	t.Helper()
	c.next++
	id := c.next

	body, err := json.Marshal(map[string]any{"id": id, "method": method, "params": params})
	if err != nil {
		t.Fatal(err)
	}
	c.write(t, body)

	for {
		var msg struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal(c.read(t), &msg); err != nil {
			t.Fatal(err)
		}
		if msg.ID != id {
			continue
		}
		if msg.Error != nil {
			t.Fatalf("%s: %s", method, msg.Error.Message)
		}
		var result map[string]any
		if len(msg.Result) > 0 {
			_ = json.Unmarshal(msg.Result, &result)
		}
		return result
	}
}

// eval runs an expression in the page and returns its value. The expressions
// are literals written in this package — this is a browser these tests
// launched, driving a server these tests started, with no input from anywhere
// else. The expression is
// wrapped so that `await` works and so a thrown error fails the test where it
// happened rather than as a mystery null three assertions later.
func (c *cdp) eval(t *testing.T, expression string) any {
	t.Helper()

	result := c.send(t, "Runtime.evaluate", map[string]any{
		"expression":    "(async () => { " + expression + " })()",
		"awaitPromise":  true,
		"returnByValue": true,
	})
	if details, ok := result["exceptionDetails"]; ok {
		t.Fatalf("the page threw: %v\n%s", details, expression)
	}
	value, _ := result["result"].(map[string]any)
	return value["value"]
}

func (c *cdp) navigate(t *testing.T, url string) {
	t.Helper()
	c.send(t, "Page.navigate", map[string]any{"url": url})
	// Waiting for the document rather than for a fixed sleep: the page is
	// ready when its own script has wired the page up, which is what every
	// assertion here is about.
	deadline := time.Now().Add(20 * time.Second)
	for {
		if c.eval(t, "return document.readyState") == "complete" {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("%s never finished loading", url)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// key dispatches a real key event through the browser's input pipeline, rather
// than a synthetic one from a script. What the page sees is what a keyboard
// would have produced.
func (c *cdp) key(t *testing.T, k string) {
	t.Helper()
	for _, kind := range []string{"keyDown", "keyUp"} {
		params := map[string]any{"type": kind, "key": k}
		if kind == "keyDown" && len(k) == 1 {
			params["text"] = k
		}
		c.send(t, "Input.dispatchKeyEvent", params)
	}
	time.Sleep(120 * time.Millisecond)
}

// until polls an expression until it is true, so a test says what it is waiting
// for instead of guessing how long to sleep.
func (c *cdp) until(t *testing.T, what, expression string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		if c.eval(t, "return await ("+expression+")") == true {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s (%s)", what, strings.TrimSpace(expression))
		}
		time.Sleep(50 * time.Millisecond)
	}
}
