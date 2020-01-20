package common

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestNewSpeedCounter(t *testing.T) {
	assert := NewAssert(t)

	sc := NewSpeedCounter()
	assert(sc.total).Equals(int64(0))
	assert(sc.lastCount).Equals(int64(0))
	assert(sc.lastNS > 0).IsTrue()
}

func TestSpeedCounter_Add(t *testing.T) {
	assert := NewAssert(t)
	sc := NewSpeedCounter()
	sc.Add(5)
	assert(sc.Total()).Equals(int64(5))
	sc.Add(-5)
	assert(sc.Total()).Equals(int64(0))
}

func TestSpeedCounter_Total(t *testing.T) {
	assert := NewAssert(t)
	sc := NewSpeedCounter()
	sc.Add(5)
	sc.Add(10)
	assert(sc.Total()).Equals(int64(15))
}

func TestSpeedCounter_Calculate(t *testing.T) {
	assert := NewAssert(t)

	// Add 100 and calculate
	sc := NewSpeedCounter()
	sc.Add(100)
	time.Sleep(100 * time.Millisecond)
	speed := sc.Calculate()
	assert(speed > 700 && speed < 1100).IsTrue()

	// Add -100 and calculate
	sc = NewSpeedCounter()
	sc.Add(-100)
	time.Sleep(100 * time.Millisecond)
	speed = sc.Calculate()
	assert(speed > -1100 && speed < -700).IsTrue()

	// time interval is zero
	sc = NewSpeedCounter()
	sc.Add(1)
	v := sc.calculate(sc.lastNS)
	assert(v).Equals(int64(0))
}

func TestSpeedCounter_Calculate_Parallels(t *testing.T) {
	sc := NewSpeedCounter()

	start := int32(1)
	fnCounter := func() {
		for atomic.LoadInt32(&start) > 0 {
			sc.Add(1)
			time.Sleep(time.Millisecond)
		}
	}
	fnCalculate := func() {
		for atomic.LoadInt32(&start) > 0 {
			sc.Calculate()
			time.Sleep(time.Millisecond)
		}
	}

	for i := 0; i < 20; i++ {
		go fnCounter()
		go fnCalculate()
	}

	time.Sleep(300 * time.Millisecond)
	atomic.StoreInt32(&start, 0)
}
