package common

import (
	"sync"
)

// AutoLock ...
type AutoLock struct {
	locker sync.Mutex
}

// CallWithLock ...
func (p *AutoLock) CallWithLock(fn func() interface{}) interface{} {
	p.locker.Lock()
	ret := fn()
	p.locker.Unlock()
	return ret
}

// DoWithLock ...
func (p *AutoLock) DoWithLock(fn func()) {
	p.locker.Lock()
	fn()
	p.locker.Unlock()
}
