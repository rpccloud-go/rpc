package common

import (
	"testing"
	"unsafe"
)

// test IsNil fail
func runReportAssertFail(fn func()) {
	ch := make(chan bool, 1)
	originReportFail := reportAssertFail
	reportAssertFail = func(p *Assert) {
		ch <- true
	}
	fn()
	reportAssertFail = originReportFail
	<-ch
}

func TestNewAssert(t *testing.T) {
	assert := NewAssert(t)

	assert(assert(3).args[0]).Equals(3)
	assert(assert(3).t).Equals(t)

	assert(assert(3, true, nil).args...).Equals(3, true, nil)
	assert(assert(3).t).Equals(t)
}

func TestAssert_Fail(t *testing.T) {
	assert := NewAssert(t)

	runReportAssertFail(func() {
		assert(nil).Fail()
	})
}

func TestAssert_Equals(t *testing.T) {
	assert := NewAssert(t)
	assert(nil).Equals(nil)
	assert(3).Equals(3)
	assert((interface{})(nil)).Equals(nil)
	assert((*Assert)(nil)).Equals((*Assert)(nil))
	assert(nil).Equals((interface{})(nil))
	assert(nil).Equals((*Assert)(nil))
	assert([]int{1, 2, 3}).Equals([]int{1, 2, 3})
	assert(map[int]string{3: "OK", 4: "NO"}).
		Equals(map[int]string{4: "NO", 3: "OK"})

	runReportAssertFail(func() {
		assert(nil).Equals(0)
	})
	runReportAssertFail(func() {
		assert(3).Equals(uint(3))
	})
	runReportAssertFail(func() {
		assert((interface{})(nil)).Equals(0)
	})
	runReportAssertFail(func() {
		assert([]int{1, 2, 3}).Equals([]int64{1, 2, 3})
	})
	runReportAssertFail(func() {
		assert([]int{1, 2, 3}).Equals([]int32{1, 2, 3})
	})
	runReportAssertFail(func() {
		assert(map[int]string{3: "OK", 4: "NO"}).
			Equals(map[int64]string{4: "NO", 3: "OK"})
	})
	runReportAssertFail(func() {
		assert().Equals(3)
	})
	runReportAssertFail(func() {
		assert(3, 3).Equals(3)
	})
}

func TestAssert_Contains(t *testing.T) {
	assert := NewAssert(t)
	assert("hello").Contains("")
	assert("hello").Contains("o")
	assert([]byte{1, 2, 3}).Contains([]byte{})
	assert([]byte{1, 2, 3}).Contains(byte(1))
	assert([]byte{1, 2, 3}).Contains([]byte{1, 2})
	assert([]byte{1, 2, 3}).Contains([]byte{2, 3})
	assert([]int{1, 2, 3}).Contains([]int{})
	assert([]int{1, 2, 3}).Contains(1)
	assert([]int{1, 2, 3}).Contains([]int{1, 2})
	assert([]int{1, 2, 3}).Contains([]int{2, 3})
	assert([]uint{1, 2, 3}).Contains([]uint{})
	assert([]uint{1, 2, 3}).Contains(uint(1))
	assert([]uint{1, 2, 3}).Contains([]uint{1, 2})
	assert([]uint{1, 2, 3}).Contains([]uint{2, 3})
	assert([0]int{}).Contains([0]int{})
	assert([3]int{1, 2, 3}).Contains(1)
	assert([3]int{1, 2, 3}).Contains([]int{1, 2})
	assert([3]int{1, 2, 3}).Contains([2]int{2, 3})

	runReportAssertFail(func() {
		assert(3).Contains(3)
	})
	runReportAssertFail(func() {
		assert(3).Contains(4)
	})
	runReportAssertFail(func() {
		assert(nil).Contains(3)
	})
	runReportAssertFail(func() {
		assert(nil).Contains(nil)
	})
	runReportAssertFail(func() {
		assert([]byte{1, 2, 3}).Contains([]byte{1, 3})
	})
	runReportAssertFail(func() {
		assert([]int{1, 2, 3}).Contains([]int{1, 3})
	})
	runReportAssertFail(func() {
		assert([]uint{1, 2, 3}).Contains([]uint{1, 3})
	})
	runReportAssertFail(func() {
		assert([]uint{}).Contains([]uint{1, 3})
	})
	runReportAssertFail(func() {
		assert("hello").Contains(3)
	})
	runReportAssertFail(func() {
		assert([]byte{1, 2, 3}).Contains(3)
	})
	runReportAssertFail(func() {
		assert([]interface{}{1, 3}).Contains(nil)
	})
	runReportAssertFail(func() {
		assert().Contains(2)
	})
}

func TestAssert_IsNil(t *testing.T) {
	assert := NewAssert(t)

	assert(nil).IsNil()
	assert((*Assert)(nil)).IsNil()
	assert((unsafe.Pointer)(nil)).IsNil()
	assert(uintptr(0)).IsNil()

	runReportAssertFail(func() {
		assert(NewAssert(t)).IsNil()
	})
	runReportAssertFail(func() {
		assert(32).IsNil()
	})
	runReportAssertFail(func() {
		assert(false).IsNil()
	})
	runReportAssertFail(func() {
		assert(0).IsNil()
	})
	runReportAssertFail(func() {
		assert().IsNil()
	})
}

func TestAssert_IsNotNil(t *testing.T) {
	assert := NewAssert(t)
	assert(t).IsNotNil()

	runReportAssertFail(func() {
		assert(nil).IsNotNil()
	})
	runReportAssertFail(func() {
		assert((*Assert)(nil)).IsNotNil()
	})
	runReportAssertFail(func() {
		assert().IsNotNil()
	})
}

func TestAssert_IsTrue(t *testing.T) {
	assert := NewAssert(t)
	assert(true).IsTrue()

	runReportAssertFail(func() {
		assert((*Assert)(nil)).IsTrue()
	})
	runReportAssertFail(func() {
		assert(32).IsTrue()
	})
	runReportAssertFail(func() {
		assert(false).IsTrue()
	})
	runReportAssertFail(func() {
		assert(0).IsTrue()
	})
	runReportAssertFail(func() {
		assert().IsTrue()
	})
}

func TestAssert_IsFalse(t *testing.T) {
	assert := NewAssert(t)
	assert(false).IsFalse()

	runReportAssertFail(func() {
		assert(32).IsFalse()
	})
	runReportAssertFail(func() {
		assert(true).IsFalse()
	})
	runReportAssertFail(func() {
		assert(0).IsFalse()
	})
	runReportAssertFail(func() {
		assert().IsFalse()
	})
}

type testAssertFail struct {
	ch chan bool
}

func (p *testAssertFail) Fail() {
	p.ch <- true
}

func Test_reportFail(t *testing.T) {
	assert := NewAssert(t)

	ch := make(chan bool, 1)
	target := NewAssert(t)(3)
	target.t = &testAssertFail{ch: ch}
	target.Equals(4)

	assert(<-target.t.(*testAssertFail).ch).IsTrue()
}
