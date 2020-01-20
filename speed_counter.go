package common

import (
	"sync"
	"sync/atomic"
	"time"
)

// SpeedCounter count the speed
type SpeedCounter struct {
	total     int64
	lastCount int64
	lastNS    int64
	sync.Mutex
}

// NewSpeedCounter create a SpeedCounter
func NewSpeedCounter() *SpeedCounter {
	return &SpeedCounter{
		total:     0,
		lastCount: 0,
		lastNS:    time.Now().UnixNano(),
	}
}

// Add count n times
func (p *SpeedCounter) Add(n int64) {
	atomic.AddInt64(&p.total, n)
}

// Total get the total count
func (p *SpeedCounter) Total() int64 {
	return atomic.LoadInt64(&p.total)
}

func (p *SpeedCounter) calculate(nowNS int64) int64 {
	p.Lock()
	defer p.Unlock()

	deltaNS := nowNS - p.lastNS
	if deltaNS <= 0 {
		return 0
	}

	deltaCount := atomic.LoadInt64(&p.total) - p.lastCount

	p.lastCount += deltaCount
	p.lastNS += deltaNS

	return (deltaCount * int64(time.Second)) / deltaNS
}

// Calculate get the speed and reinitialized the SpeedCounter
func (p *SpeedCounter) Calculate() int64 {
	return p.calculate(time.Now().UnixNano())
}
