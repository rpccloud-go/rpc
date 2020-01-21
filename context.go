package rpc

import (
	"fmt"
	"sync/atomic"
	"unsafe"
)

type rpcContext struct {
	thread unsafe.Pointer
}

func (p *rpcContext) getThread() *rpcThread {
	return (*rpcThread)(p.thread)
}

func (p *rpcContext) stop() {
	atomic.StorePointer(&p.thread, nil)
}

func (p *rpcContext) writeError(message string, debug string) *rpcReturn {
	if thread := p.getThread(); thread != nil {
		if thread.threadPool != nil &&
			thread.threadPool.processor != nil &&
			thread.threadPool.processor.logger != nil {
			thread.threadPool.processor.logger.Error(
				NewErrorByDebug(message, debug).Error(),
			)
		}
		execStream := thread.outStream
		execStream.SetWritePos(17)
		execStream.WriteBool(false)
		execStream.WriteString(message)
		execStream.WriteString(debug)
		thread.execSuccessful = false
	}
	return nilReturn
}

// OK get success Return  by value
func (p *rpcContext) OK(value interface{}) *rpcReturn {
	if thread := p.getThread(); thread != nil {
		stream := thread.outStream
		stream.SetWritePos(17)
		stream.WriteBool(true)

		if stream.Write(value) != rpcStreamWriteOK {
			return p.writeError("return type is error", GetStackString(1))
		}

		thread.execSuccessful = true
	}
	return nilReturn
}

func (p *rpcContext) Error(err Error) *rpcReturn {
	if err == nil {
		return nilReturn
	}

	if thread := p.getThread(); thread != nil &&
		thread.execEchoNode != nil &&
		thread.execEchoNode.debugString != "" {
		err.AddDebug(thread.execEchoNode.debugString)
	}

	return p.writeError(err.GetMessage(), err.GetDebug())
}

func (p *rpcContext) Errorf(format string, a ...interface{}) *rpcReturn {
	return p.Error(
		NewErrorByDebug(
			fmt.Sprintf(format, a...),
			GetStackString(1),
		))
}
