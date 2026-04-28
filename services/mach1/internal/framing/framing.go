// Package framing implements newline-delimited JSON framing used by MCP stdio
// transport. Each message is a single JSON object terminated by '\n'.
//
// We use bufio.Scanner with a generous buffer because MCP responses (e.g.
// tools/list with rich schemas) regularly exceed the default 64KiB limit.
package framing

import (
	"bufio"
	"encoding/json"
	"io"
	"sync"
)

// MaxMessageBytes caps a single inbound message. 8 MiB is well above any
// realistic MCP payload and protects against runaway upstreams.
const MaxMessageBytes = 8 << 20

// Reader yields one raw JSON message per Read call.
type Reader struct {
	sc *bufio.Scanner
}

func NewReader(r io.Reader) *Reader {
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64<<10), MaxMessageBytes)
	sc.Split(bufio.ScanLines)
	return &Reader{sc: sc}
}

// Read returns the next message bytes (without the trailing newline) or
// io.EOF. The returned slice is owned by the scanner and must be copied if
// retained beyond the next Read.
func (r *Reader) Read() ([]byte, error) {
	if r.sc.Scan() {
		return r.sc.Bytes(), nil
	}
	if err := r.sc.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// Writer serializes concurrent writes to an io.Writer with a single mutex so
// interleaved goroutines never produce torn JSON lines.
type Writer struct {
	mu sync.Mutex
	w  io.Writer
}

func NewWriter(w io.Writer) *Writer { return &Writer{w: w} }

// Write marshals v and emits it as a single newline-terminated line.
func (w *Writer) Write(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err = w.w.Write([]byte{'\n'})
	return err
}

// WriteRaw emits a pre-marshaled JSON message verbatim with newline framing.
func (w *Writer) WriteRaw(b []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, err := w.w.Write(b); err != nil {
		return err
	}
	_, err := w.w.Write([]byte{'\n'})
	return err
}
