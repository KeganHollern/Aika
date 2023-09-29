package utils

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStringPipe(t *testing.T) {
	assert := assert.New(t)
	sp := NewStringPipe()

	var wg sync.WaitGroup
	wg.Add(1)

	// Consumer: Reads from StringPipe
	go func() {
		defer wg.Done()
		for i := 0; i < 2; i++ {
			str, err := sp.Read()
			assert.Nil(err, "Unexpected read error")
			assert.Equal("data", str, "Expected 'data'")
		}
	}()

	// Producer: Writes to StringPipe character by character
	go func() {
		for i := 0; i < 10; i++ {
			for _, ch := range "data\n" {
				_, err := sp.Write([]byte{byte(ch)})
				assert.Nil(err, "Unexpected write error")
				time.Sleep(time.Millisecond)
			}
		}
		sp.Close()
	}()

	wg.Wait()
}

func TestWrite(t *testing.T) {
	sp := NewStringPipe()
	n, err := sp.Write([]byte("hello\n"))

	assert.Nil(t, err)
	assert.Equal(t, 6, n)
}

func TestRead(t *testing.T) {
	sp := NewStringPipe()
	go func() {
		sp.Write([]byte("hello\nworld\n"))
	}()

	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "hello", line)

	line, err = sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "world", line)
}

func TestRead2(t *testing.T) {
	sp := NewStringPipe()
	go func() {
		sp.Write([]byte(" \n\nhello\nworld\n"))
	}()
	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, " ", line)

	line, err = sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "", line)

	line, err = sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "hello", line)

	line, err = sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "world", line)
}

func TestClose(t *testing.T) {
	sp := NewStringPipe()

	err := sp.Close()
	assert.Nil(t, err)

	_, err = sp.Write([]byte("should fail\n"))
	assert.Equal(t, io.ErrClosedPipe, err)

	_, err = sp.Read()
	assert.Equal(t, io.EOF, err)

	err = sp.Close()
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestConcurrentWriteRead(t *testing.T) {
	sp := NewStringPipe()
	go func() {
		time.Sleep(time.Millisecond * 50)
		sp.Write([]byte("concurrent\n"))
	}()

	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "concurrent", line)
}

func TestCloseFlushesRemainingCharacters(t *testing.T) {
	sp := NewStringPipe()
	go func() {
		sp.Write([]byte("flush"))
		sp.Close()
	}()

	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "flush", line)

	_, err = sp.Read()
	assert.Equal(t, io.EOF, err)
}
