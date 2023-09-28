package utils

import (
	"bytes"
	"io"
	"sync"
)

type StringPipe struct {
	buffer   bytes.Buffer
	dataCond *sync.Cond
	closed   bool
}

var _ io.WriteCloser = &StringPipe{}

func NewStringPipe() *StringPipe {
	return &StringPipe{
		dataCond: sync.NewCond(&sync.Mutex{}),
	}
}

func (sp *StringPipe) Write(p []byte) (n int, err error) {
	sp.dataCond.L.Lock()
	defer sp.dataCond.L.Unlock()

	if sp.closed {
		return 0, io.ErrClosedPipe
	}

	n, err = sp.buffer.Write(p)

	// Signal only when newline is encountered in the input buffer.
	if bytes.Contains(p, []byte{'\n'}) {
		sp.dataCond.Signal()
	}

	return
}

// reads line by line
func (sp *StringPipe) Read() (string, error) {
	sp.dataCond.L.Lock()
	defer sp.dataCond.L.Unlock()

	for {
		line, err := sp.buffer.ReadString('\n')

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

	sp.closed = true
	sp.dataCond.Signal()
	return nil
}
