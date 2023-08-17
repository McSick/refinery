package buffer

import (
	"sync"
	"github.com/honeycombio/refinery/types"
	"fmt"
	"time"
	"strings"
)
type EventMetadata struct {
    RetryCount    int
    NextRetryTime time.Duration
}

type ExtendedEvent struct {
    *types.Event
    Metadata EventMetadata
}


type RingBuffer struct {
    mu    sync.RWMutex
    size  int
    data  []*types.Event
    start int // points to the oldest element
    end   int // points to the next empty spot
}
func NewRingBuffer(size int) *RingBuffer {
    return &RingBuffer{
        size: size,
        data: make([]*types.Event, size),
    }
}

func (rb *RingBuffer) Push(event *types.Event) {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    rb.data[rb.end] = event
    rb.end = (rb.end + 1) % rb.size

    if rb.end == rb.start {
        rb.start = (rb.start + 1) % rb.size // overwrite the oldest data
    }
}

func (rb *RingBuffer) GetAll() []*types.Event {
    rb.mu.Lock()
    defer rb.mu.Unlock()
    
    if rb.start < rb.end {
        return rb.data[rb.start:rb.end]
    }
    return append(rb.data[rb.start:], rb.data[:rb.end]...)
}

func (rb *RingBuffer) String() string {
	rb.mu.RLock()
	defer rb.mu.RUnlock()

	var bufferContents []string
	for _, item := range rb.data {
		if item != nil {
			bufferContents = append(bufferContents, fmt.Sprintf("%v", item))
		}
	}

	return fmt.Sprintf("RingBuffer Contents: [%s]", strings.Join(bufferContents, ", "))
}
