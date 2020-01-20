package rpc

import (
	"math"
	"testing"
	"time"
)

func TestPerformanceIndicator_basic(t *testing.T) {
	assert := NewAssert(t)
	performanceIndicator := NewPerformanceIndicator()

	for i := 0; i < 1500; i++ {
		go func(idx int) {
			time.Sleep(50 * time.Millisecond)
			for k := 0; k < 20; k++ {
				performanceIndicator.Count(
					time.Duration(idx*2)*time.Millisecond,
					"#",
					true,
				)
				performanceIndicator.Count(
					time.Duration(idx*2)*time.Millisecond,
					"#",
					false,
				)
				time.Sleep(100 * time.Millisecond)
			}
		}(i)
	}

	for n := 0; n < 20; n++ {
		time.Sleep(100 * time.Millisecond)
		qps, duration := performanceIndicator.Calculate(TimeNowNS())
		count := qps * int64(duration) / int64(time.Second)
		assert(math.Abs(float64(count-3000)) < 50).IsTrue()
	}

	time.Sleep(200 * time.Millisecond)

	nowNS := TimeNowNS()
	performanceIndicator.Calculate(nowNS)
	assert(performanceIndicator.Calculate(nowNS)).
		Equals(int64(0), time.Duration(0))

	assert(performanceIndicator.failed).Equals(int64(30000))
	assert(performanceIndicator.successArray).Equals([10]int64{
		20, 40, 40, 100, 200, 200, 400, 2000, 7000, 20000,
	})
	assert(performanceIndicator.lastTotal).Equals(int64(60000))
	assert(performanceIndicator.lastNS).Equals(nowNS)
}
