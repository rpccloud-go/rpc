package rpc

import (
	"sync"
)

// rpcAutoLock ...
type rpcAutoLock struct {
	locker sync.Mutex
}

// CallWithLock ...
func (p *rpcAutoLock) CallWithLock(fn func() interface{}) interface{} {
	p.locker.Lock()
	ret := fn()
	p.locker.Unlock()
	return ret
}

// DoWithLock ...
func (p *rpcAutoLock) DoWithLock(fn func()) {
	p.locker.Lock()
	fn()
	p.locker.Unlock()
}
