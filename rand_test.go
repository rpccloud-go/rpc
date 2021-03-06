package rpc

import (
	"testing"
)

func TestGetRandUint32(t *testing.T) {
	assert := newAssert(t)
	sum := int64(0)
	for i := 0; i < 100000; i++ {
		sum += int64(getRandUint32())
	}
	delta := sum/100000 - 2147483648
	assert(delta > -20000000 && delta < 20000000).IsTrue()
}

func TestGetRandString(t *testing.T) {
	assert := newAssert(t)
	assert(getRandString(-1)).Equals("")
	for i := 0; i < 100; i++ {
		assert(len(getRandString(i))).Equals(i)
	}
}

func BenchmarkGetRandUint32(b *testing.B) {
	b.ReportAllocs()
	for n := 0; n < b.N; n++ {
		getRandString(128)
	}
}
