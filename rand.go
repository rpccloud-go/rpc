package rpc

import (
	"math/rand"
	"sync/atomic"
	"time"
)

const (
	randCacheSize = 1000001
	base64String  = "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789" +
		"+/"
)

var (
	randPos         = uint64(0)
	randCacheUint32 = make([]uint32, randCacheSize, randCacheSize)
)

// getRandUint32 get Uint32 rand and fast,
// this is approximate rand but high performance
func getRandUint32() uint32 {
	return randCacheUint32[atomic.AddUint64(&randPos, 1)%randCacheSize]
}

// getRandString get random string
func getRandString(strLen int) string {
	sb := NewStringBuilder()
	for strLen > 0 {
		rand64 := rand.Uint64()
		for used := 0; used < 10 && strLen > 0; used++ {
			sb.AppendByte(base64String[rand64%64])
			rand64 = rand64 / 64
			strLen--
		}
	}
	ret := sb.String()
	sb.Release()
	return ret
}

func init() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < randCacheSize; i++ {
		randCacheUint32[i] = rand.Uint32()
	}
}
