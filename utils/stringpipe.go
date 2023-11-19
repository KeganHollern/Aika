package utils

import (
	"bytes"
	"io"
	"sync"
)

type StringPipe struct {
	buffer   bytes.Buffer
	temp     bytes.Buffer
	dataCond *sync.Cond
	closed   bool
	splitter byte
}

// TODO: implement io.Reader interface so it matches
// the bytepipe implemented
var _ io.WriteCloser = &StringPipe{}

func NewStringPipe(delim byte) *StringPipe {
	return &StringPipe{
		dataCond: sync.NewCond(&sync.Mutex{}),
		splitter: delim,
	}
}

func (sp *StringPipe) Write(p []byte) (n int, err error) {
	sp.dataCond.L.Lock()
	defer sp.dataCond.L.Unlock()

	if sp.closed {
		return 0, io.ErrClosedPipe
	}

	// Write to a temporary buffer first
	n, err = sp.temp.Write(p)

	// Check for newline in the temporary buffer
	if bytes.Contains(sp.temp.Bytes(), []byte{sp.splitter}) {
		sp.buffer.Write(sp.temp.Bytes())
		sp.temp.Reset()
		sp.dataCond.Signal()
	}

	return
}

// reads line by line
func (sp *StringPipe) Read() (string, error) {
	sp.dataCond.L.Lock()
	defer sp.dataCond.L.Unlock()

	for {
		line, err := sp.buffer.ReadString(sp.splitter)

		if err == io.EOF {
			if sp.closed {
				if len(line) > 0 {
					return line, nil
				}
				return "", io.EOF
			}
			sp.dataCond.Wait()
			continue
		}

		// Remove the trailing newline
		line = line[:len(line)-1]

		return line, nil
	}
}

func (sp *StringPipe) Close() error {
	sp.dataCond.L.Lock()
	defer sp.dataCond.L.Unlock()

	if sp.closed {
		return io.ErrClosedPipe
	}

	// Append a newline to flush any remaining characters.
	if sp.temp.Len() > 0 {
		sp.temp.Write([]byte{sp.splitter})
		sp.buffer.Write(sp.temp.Bytes())
		sp.temp.Reset()
	}

	sp.closed = true
	sp.dataCond.Signal()
	return nil
}
