package utils

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentReadWrites_BP(t *testing.T) {
	bp := NewBytePipe()
	var wg sync.WaitGroup

	const iterations = 1000
	const goroutines = 10
	data := []byte("test")

	wg.Add(goroutines * 2) // 10 for writing, 10 for reading

	// Concurrent Writes
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, err := bp.Write(data)
				assert.NoError(t, err)
			}
			wg.Done()
		}()
	}

	// Concurrent Reads
	readBuffer := make([]byte, 4)
	for i := 0; i < goroutines; i++ {
		go func() {
			for j := 0; j < iterations; j++ {
				_, err := bp.Read(readBuffer)
				assert.NoError(t, err)
				assert.Equal(t, data, readBuffer)
			}
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestNewBytePipe_BP(t *testing.T) {
	bp := NewBytePipe()
	assert.NotNil(t, bp)
	assert.NotNil(t, bp.cond)
}

func TestWrite_BP(t *testing.T) {
	bp := NewBytePipe()
	data := []byte("test")

	n, err := bp.Write(data)

	assert.NoError(t, err)
	assert.Equal(t, 4, n)
}

func TestWriteClosedBuffer_BP(t *testing.T) {
	bp := NewBytePipe()
	bp.Close()

	n, err := bp.Write([]byte("test"))

	assert.Error(t, err)
	assert.Equal(t, 0, n)
}

func TestRead_BP(t *testing.T) {
	bp := NewBytePipe()
	data := []byte("test")
	readBuffer := make([]byte, 4)

	go func() {
		time.Sleep(100 * time.Millisecond)
		bp.Write(data)
	}()

	n, err := bp.Read(readBuffer)

	assert.NoError(t, err)
	assert.Equal(t, 4, n)
	assert.Equal(t, data, readBuffer)
}

func TestReadClosedBuffer_BP(t *testing.T) {
	bp := NewBytePipe()
	readBuffer := make([]byte, 4)

	bp.Close()
	n, err := bp.Read(readBuffer)

	assert.Equal(t, io.EOF, err)
	assert.Equal(t, 0, n)
}

func TestClose_BP(t *testing.T) {
	bp := NewBytePipe()
	err := bp.Close()

	assert.NoError(t, err)
	assert.True(t, bp.closed)
}
