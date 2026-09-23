package audio

import (
	"bytes"
	"errors"
	"io"
	"testing"
	"time"
)

// slowReader hands out data in small pieces with a pause between them, like
// a congested network share.
type slowReader struct {
	data  []byte
	step  int
	pause time.Duration
}

func (s *slowReader) Read(p []byte) (int, error) {
	if len(s.data) == 0 {
		return 0, io.EOF
	}
	time.Sleep(s.pause)
	n := copy(p[:min(len(p), s.step)], s.data)
	s.data = s.data[n:]
	return n, nil
}

func (s *slowReader) Close() error { return nil }

func testBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(i * 7)
	}
	return b
}

func TestMemFileReadsWhileLoading(t *testing.T) {
	want := testBytes(50_000)
	m := startLoad("x.flac", &slowReader{data: want, step: 4096, pause: time.Millisecond}, int64(len(want)))

	r1, _ := m.open()
	r2, _ := m.open()
	got1, err := io.ReadAll(r1)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got1, want) {
		t.Fatal("reader 1 saw different bytes")
	}

	// A second reader is independent and can seek relative to the end,
	// using the size known before loading finished.
	if _, err := r2.Seek(-100, io.SeekEnd); err != nil {
		t.Fatal(err)
	}
	tail, _ := io.ReadAll(r2)
	if !bytes.Equal(tail, want[len(want)-100:]) {
		t.Error("seek from end returned wrong bytes")
	}
}

func TestMemFileReportsLoadErrors(t *testing.T) {
	m := startLoad("x.flac", &failingReader{}, 10)
	r, _ := m.open()
	if _, err := io.ReadAll(r); err == nil || !errors.Is(err, errBroken) {
		t.Fatalf("ReadAll error = %v, want errBroken", err)
	}
}

var errBroken = errors.New("share went away")

type failingReader struct{ sent bool }

func (f *failingReader) Read(p []byte) (int, error) {
	if !f.sent {
		f.sent = true
		return copy(p, "abc"), nil
	}
	return 0, errBroken
}

func (f *failingReader) Close() error { return nil }

func TestTrackCacheKeepsOnlyRequested(t *testing.T) {
	loads := map[string]int{}
	c := newTrackCache()
	c.load = func(path string) (*memFile, error) {
		loads[path]++
		return startLoad(path, io.NopCloser(bytes.NewReader(nil)), 0), nil
	}

	tests := []struct {
		name  string
		step  func()
		want  []string
		loads map[string]int
	}{
		{name: "play a", step: func() { c.keep("a"); _, _ = c.get("a") }, want: []string{"a"}, loads: map[string]int{"a": 1}},
		{name: "preload b", step: func() { c.keep("a", "b"); _, _ = c.get("b") }, want: []string{"a", "b"}, loads: map[string]int{"a": 1, "b": 1}},
		{name: "play preloaded b", step: func() { c.keep("b"); _, _ = c.get("b") }, want: []string{"b"}, loads: map[string]int{"a": 1, "b": 1}},
		{name: "stop", step: func() { c.keep() }, want: nil, loads: map[string]int{"a": 1, "b": 1}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.step()
			for _, p := range []string{"a", "b"} {
				if c.has(p) != contains(tc.want, p) {
					t.Errorf("has(%s) = %v", p, c.has(p))
				}
				if loads[p] != tc.loads[p] {
					t.Errorf("loads[%s] = %d, want %d", p, loads[p], tc.loads[p])
				}
			}
		})
	}
}
