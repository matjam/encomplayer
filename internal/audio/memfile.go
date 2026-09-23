package audio

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
)

// loadChunk is how much the loader reads per call. Large reads keep the
// number of network round trips low on SMB and NFS shares.
const loadChunk = 1 << 20

// memFile is a file being read into memory by a background goroutine.
// Readers see bytes as soon as they arrive and wait only when they get
// ahead of the loader, so decoding never touches the disk or network.
type memFile struct {
	path string
	size int64

	mu       sync.Mutex
	cond     *sync.Cond
	data     []byte
	done     bool
	err      error
	canceled bool
}

// loadFile starts reading path into memory.
func loadFile(path string) (*memFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("load %s: %w", path, err)
	}
	return startLoad(path, f, info.Size()), nil
}

// startLoad reads r, of the given size, into memory in the background and
// closes it when done.
func startLoad(path string, r io.ReadCloser, size int64) *memFile {
	m := &memFile{path: path, size: size, data: make([]byte, 0, size)}
	m.cond = sync.NewCond(&m.mu)
	go m.load(r)
	return m
}

func (m *memFile) load(r io.ReadCloser) {
	defer r.Close()
	buf := make([]byte, loadChunk)
	for {
		n, err := r.Read(buf)

		m.mu.Lock()
		m.data = append(m.data, buf[:n]...)
		stop := m.canceled
		switch {
		case errors.Is(err, io.EOF):
			m.done = true
		case err != nil:
			m.done, m.err = true, fmt.Errorf("load %s: %w", m.path, err)
		case stop:
			m.done, m.err = true, errors.New("load canceled")
		}
		finished := m.done
		m.cond.Broadcast()
		m.mu.Unlock()

		if finished {
			return
		}
	}
}

// cancel stops loading. The buffer is freed once nothing references the
// file.
func (m *memFile) cancel() {
	m.mu.Lock()
	m.canceled = true
	m.mu.Unlock()
}

// open returns an independent reader positioned at the start.
func (m *memFile) open() (io.ReadSeekCloser, error) {
	return &memReader{file: m}, nil
}

// memReader reads a memFile.
type memReader struct {
	file *memFile
	off  int64
}

func (r *memReader) Read(p []byte) (int, error) {
	m := r.file
	m.mu.Lock()
	defer m.mu.Unlock()

	for r.off >= int64(len(m.data)) && !m.done {
		m.cond.Wait()
	}
	if r.off >= int64(len(m.data)) {
		if m.err != nil {
			return 0, m.err
		}
		return 0, io.EOF
	}
	n := copy(p, m.data[r.off:])
	r.off += int64(n)
	return n, nil
}

func (r *memReader) Seek(offset int64, whence int) (int64, error) {
	var base int64
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		base = r.off
	case io.SeekEnd:
		base = r.file.size
	default:
		return 0, fmt.Errorf("seek %s: invalid whence %d", r.file.path, whence)
	}
	if base+offset < 0 {
		return 0, fmt.Errorf("seek %s: negative position", r.file.path)
	}
	r.off = base + offset
	return r.off, nil
}

// Close is a no-op; the player owns the memFile's lifetime.
func (r *memReader) Close() error { return nil }
