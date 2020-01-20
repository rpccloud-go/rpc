package rpc

import "testing"

func TestAutoLock_CallWithLock(t *testing.T) {
	assert := newAssert(t)
	locker := AutoLock{}
	waits := make(chan bool)
	sum := 0

	for i := 0; i < 100; i++ {
		go func() {
			for n := 0; n < 1000; n++ {
				assert(locker.CallWithLock(func() interface{} {
					sum += n
					return true
				})).Equals(true)
			}
			waits <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-waits
	}

	assert(sum).Equals(49950000)
}

func TestAutoLock_DoWithLock(t *testing.T) {
	assert := newAssert(t)
	locker := AutoLock{}
	waits := make(chan bool)
	sum := 0

	for i := 0; i < 100; i++ {
		go func() {
			for n := 0; n < 1000; n++ {
				locker.DoWithLock(func() {
					sum += n
				})
			}
			waits <- true
		}()
	}

	for i := 0; i < 100; i++ {
		<-waits
	}

	assert(sum).Equals(49950000)
}
