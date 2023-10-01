package transcoding

import "sync"

type Mixer struct {
	channels []chan []int16
	mu       sync.Mutex
	out      chan []int16
	wg       sync.WaitGroup
	quit     chan struct{}
}

// Create a new audio mixer outputting to the
// provided PCM channel
func NewMixer(output chan []int16) *Mixer {
	return &Mixer{
		channels: make([]chan []int16, 0),
		out:      output,
		quit:     make(chan struct{}),
	}
}

// Add a source to the mixer
func (m *Mixer) Add(ch chan []int16) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.channels = append(m.channels, ch)
}

// create a pcm source channel for this mixer
// close this channel once you're done with it!
func (m *Mixer) Create() chan []int16 {
	ch := make(chan []int16)
	m.Add(ch)
	return ch
}

// Start mixing - blocking
// Call STOP() to safely stop before closing
// the output channel!
func (m *Mixer) Start() {
	m.wg.Add(1)
	defer m.wg.Done()

	for {
		select {
		case <-m.quit:
			return
		default:
			if m.empty() {
				continue // no sources
			}
			data := m.merge()
			if data != nil {
				m.out <- data
			}
		}

	}
}

// Stop the mixer - blocks until
// no longer using the output channel
func (m *Mixer) Stop() {
	close(m.quit)
	m.wg.Wait()
}

func (m *Mixer) empty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return len(m.channels) == 0
}

func (m *Mixer) merge() []int16 {
	m.mu.Lock()
	defer m.mu.Unlock()

	var merged []int16
	new_channels := make([]chan []int16, 0)
	for _, ch := range m.channels {
		select {
		case data, ok := <-ch:
			if !ok {
				continue // drop this channel
			}
			// merge data with other data
			if merged == nil {
				merged = make([]int16, len(data))
			}
			for i := range merged {
				// ADD: (another operation is AVERAGE) but ADD is better for our case
				merged[i] += data[i]
			}
		default:
			// no new data on this channel to merge
		}
		new_channels = append(new_channels, ch)
	}

	m.channels = new_channels
	return merged
}
