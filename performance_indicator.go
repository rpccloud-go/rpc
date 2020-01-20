package common

import (
	"sync"
	"sync/atomic"
	"time"
)

// PerformanceIndicator ...
type PerformanceIndicator struct {
	failed       int64
	successArray [10]int64
	lastTotal    int64
	lastNS       int64
	originMap    sync.Map
}

// NewPerformanceIndicator ...
func NewPerformanceIndicator() *PerformanceIndicator {
	return &PerformanceIndicator{
		failed:       0,
		successArray: [10]int64{},
		lastTotal:    0,
		lastNS:       TimeNowNS(),
		originMap:    sync.Map{},
	}
}

// Calculate ...
func (p *PerformanceIndicator) Calculate(nowNS int64) (int64, time.Duration) {
	// calculate total called
	total := atomic.LoadInt64(&p.failed)
	for i := 0; i < 10; i++ {
		total += atomic.LoadInt64(&p.successArray[i])
	}

	// calculate delta count and time
	deltaCount := total - p.lastTotal
	deltaNS := nowNS - p.lastNS

	if deltaNS > 0 {
		p.lastTotal = total
		p.lastNS = nowNS
		return (deltaCount * int64(time.Second)) / deltaNS, time.Duration(deltaNS)
	}

	return 0, time.Duration(deltaNS)
}

// Count ...
func (p *PerformanceIndicator) Count(
	duration time.Duration,
	origin string,
	successful bool,
) {
	if successful {
		if duration < 2*time.Millisecond {
			atomic.AddInt64(&p.successArray[0], 1)
		} else if duration < 6*time.Millisecond {
			atomic.AddInt64(&p.successArray[1], 1)
		} else if duration < 10*time.Millisecond {
			atomic.AddInt64(&p.successArray[2], 1)
		} else if duration < 20*time.Millisecond {
			atomic.AddInt64(&p.successArray[3], 1)
		} else if duration < 40*time.Millisecond {
			atomic.AddInt64(&p.successArray[4], 1)
		} else if duration < 60*time.Millisecond {
			atomic.AddInt64(&p.successArray[5], 1)
		} else if duration < 100*time.Millisecond {
			atomic.AddInt64(&p.successArray[6], 1)
		} else if duration < 300*time.Millisecond {
			atomic.AddInt64(&p.successArray[7], 1)
		} else if duration < 1000*time.Millisecond {
			atomic.AddInt64(&p.successArray[8], 1)
		} else {
			atomic.AddInt64(&p.successArray[9], 1)
		}
	} else {
		atomic.AddInt64(&p.failed, 1)
	}

	if _, ok := p.originMap.Load(origin); !ok {
		p.originMap.Store(origin, true)
	}
}
