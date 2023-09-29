package utils

import (
	"bytes"
	"errors"
	"io"
	"sync"
)

// ConcurrentBuffer wraps a bytes.Buffer to make it safe for concurrent use.
type BytePipe struct {
	buf    bytes.Buffer
	mu     sync.Mutex
	closed bool
	cond   *sync.Cond
}

// NewConcurrentBuffer initializes a new ConcurrentBuffer.
func NewBytePipe() *BytePipe {
	bp := &BytePipe{}
	bp.cond = sync.NewCond(&bp.mu)
	return bp
}

// Write writes bytes to the buffer.
func (bp *BytePipe) Write(p []byte) (n int, err error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	if bp.closed {
		return 0, errors.New("buffer closed")
	}
	n, err = bp.buf.Write(p)
	bp.cond.Broadcast()
	return
}

// Read reads bytes from the buffer.
func (bp *BytePipe) Read(p []byte) (n int, err error) {
	bp.mu.Lock()
	defer bp.mu.Unlock()

	for bp.buf.Len() == 0 && !bp.closed {
		bp.cond.Wait()
	}

	if bp.buf.Len() == 0 {
		if bp.closed {
			return 0, io.EOF
		}
		return 0, nil
	}
	return bp.buf.Read(p)
}

// Close closes the buffer, after which all calls to Read will return EOF.
func (bp *BytePipe) Close() error {
	bp.mu.Lock()
	defer bp.mu.Unlock()
	bp.closed = true
	bp.cond.Broadcast()
	return nil
}
