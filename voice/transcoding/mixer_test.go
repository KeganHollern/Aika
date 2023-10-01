package transcoding

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMixerBasic(t *testing.T) {
	output := make(chan []int16, 10)
	m := NewMixer(output)

	assert.NotNil(t, m)
	assert.Equal(t, 0, len(m.channels))
}

func TestMixerAdd(t *testing.T) {
	output := make(chan []int16, 10)
	m := NewMixer(output)

	ch := make(chan []int16)
	m.Add(ch)
	assert.Equal(t, 1, len(m.channels))
}

func TestMixerStartStop(t *testing.T) {
	output := make(chan []int16, 10)
	m := NewMixer(output)

	go m.Start()

	ch1 := make(chan []int16, 10)
	ch1 <- []int16{1, 2, 3}
	m.Add(ch1)

	ch2 := make(chan []int16, 10)
	ch2 <- []int16{1, 2, 3}
	m.Add(ch2)

	assert.Equal(t, 2, len(m.channels))

	// Allow some time for goroutines to run
	// Replace with a more deterministic approach in a real-world scenario
	merged := <-output

	assert.NotNil(t, merged)
	assert.Equal(t, []int16{2, 4, 6}, merged)

	m.Stop()
	assert.Equal(t, 0, len(output))
}

func TestMixerEmpty(t *testing.T) {
	output := make(chan []int16, 10)
	m := NewMixer(output)

	assert.True(t, m.empty())

	ch := make(chan []int16)
	m.Add(ch)

	assert.False(t, m.empty())
}

func TestMixerMerge(t *testing.T) {
	output := make(chan []int16, 10)
	m := NewMixer(output)

	ch1 := make(chan []int16, 10)
	ch1 <- []int16{1, 2, 3}
	m.Add(ch1)

	ch2 := make(chan []int16, 10)
	ch2 <- []int16{1, 2, 3}
	m.Add(ch2)

	merged := m.merge()

	assert.NotNil(t, merged)
	assert.Equal(t, []int16{2, 4, 6}, merged)
}
