package utils

import (
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testChar = "|"

func TestStringPipe(t *testing.T) {
	assert := assert.New(t)
	sp := NewStringPipe(testChar[0])

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
			for _, ch := range "data" + testChar {
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
	sp := NewStringPipe(testChar[0])
	n, err := sp.Write([]byte("hello" + testChar))

	assert.Nil(t, err)
	assert.Equal(t, 6, n)
}

func TestRead(t *testing.T) {
	sp := NewStringPipe(testChar[0])
	go func() {
		sp.Write([]byte("hello" + testChar + "world" + testChar))
	}()

	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "hello", line)

	line, err = sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "world", line)
}

func TestRead2(t *testing.T) {
	sp := NewStringPipe(testChar[0])
	go func() {
		sp.Write([]byte(" " + testChar + "" + testChar + "hello" + testChar + "world" + testChar + ""))
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
	sp := NewStringPipe(testChar[0])

	err := sp.Close()
	assert.Nil(t, err)

	_, err = sp.Write([]byte("should fail" + testChar))
	assert.Equal(t, io.ErrClosedPipe, err)

	_, err = sp.Read()
	assert.Equal(t, io.EOF, err)

	err = sp.Close()
	assert.Equal(t, io.ErrClosedPipe, err)
}

func TestConcurrentWriteRead(t *testing.T) {
	sp := NewStringPipe(testChar[0])
	go func() {
		time.Sleep(time.Millisecond * 50)
		sp.Write([]byte("concurrent" + testChar))
	}()

	line, err := sp.Read()
	assert.Nil(t, err)
	assert.Equal(t, "concurrent", line)
}

func TestCloseFlushesRemainingCharacters(t *testing.T) {
	sp := NewStringPipe(testChar[0])
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
