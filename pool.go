package common

import (
	"time"
)

// rpcThreadPool
type rpcThreadPool struct {
	isRunning   bool
	processor   *RPCProcessor
	threads     []*rpcThread
	freeThreads chan *rpcThread
	AutoLock
}

func newThreadPool(processor *RPCProcessor) *rpcThreadPool {
	ret := &rpcThreadPool{
		isRunning: true,
		processor: processor,
		threads: make(
			[]*rpcThread,
			numOfThreadPerThreadPool,
			numOfThreadPerThreadPool,
		),
		freeThreads: make(chan *rpcThread, numOfThreadPerThreadPool),
	}

	for i := 0; i < numOfThreadPerThreadPool; i++ {
		thread := newThread(ret)
		ret.threads[i] = thread
		ret.freeThreads <- thread
	}

	return ret
}

func (p *rpcThreadPool) stop() bool {
	return p.CallWithLock(func() interface{} {
		if p.isRunning {
			closeCH := make(chan bool)
			// stop threads
			for i := 0; i < len(p.threads); i++ {
				go func(idx int) {
					p.threads[idx].stop()
					p.threads[idx] = nil
					closeCH <- true
				}(i)
			}

			for i := 0; i < len(p.threads); i++ {
				select {
				case <-closeCH:
				case <-time.After(5 * time.Second):
					if p.processor != nil &&
						p.processor.logger != nil {
						p.processor.logger.Error("rpc-thread-pool: internal error")
					}
				}
				select {
				case <-p.freeThreads:
				case <-time.After(5 * time.Second):
					if p.processor != nil &&
						p.processor.logger != nil {
						p.processor.logger.Error("rpc-thread-pool: internal error")
					}
				}
			}

			close(p.freeThreads)
			p.isRunning = false
			return true
		}

		return false
	}).(bool)
}

func (p *rpcThreadPool) allocThread() *rpcThread {
	return <-p.freeThreads
}

func (p *rpcThreadPool) freeThread(thread *rpcThread) {
	p.freeThreads <- thread
}
