package rpc

import (
	"testing"
	"unsafe"
)

func TestRpcContext_getThread(t *testing.T) {
	assert := NewAssert(t)

	ctx1 := rpcContext{}
	assert(ctx1.getThread()).IsNil()

	thread := newThread(nil)
	thread.stop()

	ctx2 := rpcContext{thread: unsafe.Pointer(thread)}
	assert(ctx2.getThread()).Equals(thread)

}

func TestRpcContext_stop(t *testing.T) {
	assert := NewAssert(t)

	thread := newThread(nil)
	thread.stop()

	ctx := rpcContext{thread: unsafe.Pointer(thread)}
	assert(ctx.getThread()).IsNotNil()
	ctx.stop()
	assert(ctx.getThread()).IsNil()
}

func TestRpcContext_OK(t *testing.T) {
	assert := NewAssert(t)

	// ctx is ok
	thread := newThread(nil)
	thread.stop()
	assert(thread.execSuccessful).IsFalse()
	ctx := rpcContext{thread: unsafe.Pointer(thread)}
	ctx.OK(uint(215))
	assert(thread.execSuccessful).IsTrue()
	thread.outStream.SetReadPos(17)
	assert(thread.outStream.ReadBool()).Equals(true, true)
	assert(thread.outStream.ReadUint64()).Equals(Uint(215), true)
	assert(thread.outStream.CanRead()).IsFalse()

	// ctx is stop
	thread1 := newThread(nil)
	thread1.stop()
	assert(thread1.execSuccessful).IsFalse()
	ctx1 := rpcContext{thread: unsafe.Pointer(thread1)}
	ctx1.stop()
	ctx1.OK(uint(215))
	thread1.outStream.SetReadPos(17)
	assert(thread1.outStream.GetWritePos()).Equals(17)
	assert(thread1.outStream.GetReadPos()).Equals(17)

	// value is illegal
	thread2 := newThread(nil)
	thread2.stop()
	assert(thread2.execSuccessful).IsFalse()
	ctx2 := rpcContext{thread: unsafe.Pointer(thread2)}
	ctx2.OK(make(chan bool))
	assert(thread2.execSuccessful).IsFalse()
	thread2.outStream.SetReadPos(17)
	assert(thread2.outStream.ReadBool()).Equals(false, true)
	assert(thread2.outStream.ReadString()).Equals("return type is error", true)
	dbgMessage, ok := thread2.outStream.ReadString()
	assert(ok).IsTrue()
	assert(dbgMessage).Contains("TestRpcContext_OK")
	assert(thread2.outStream.CanRead()).IsFalse()
}

func TestRpcContext_writeError(t *testing.T) {
	assert := NewAssert(t)

	// ctx is ok
	processor := NewRPCProcessor(NewLogger(), 16, 16, nil, nil)
	thread := newThread(newThreadPool(processor))
	thread.stop()
	thread.execSuccessful = true
	ctx := rpcContext{thread: unsafe.Pointer(thread)}
	ctx.writeError("errorMessage", "errorDebug")
	assert(thread.execSuccessful).IsFalse()
	thread.outStream.SetReadPos(17)
	assert(thread.outStream.ReadBool()).Equals(false, true)
	assert(thread.outStream.ReadString()).Equals("errorMessage", true)
	assert(thread.outStream.ReadString()).Equals("errorDebug", true)
	assert(thread.outStream.CanRead()).IsFalse()

	// ctx is stop
	thread1 := newThread(nil)
	thread1.stop()
	thread1.execSuccessful = true
	ctx1 := rpcContext{thread: unsafe.Pointer(thread1)}
	ctx1.stop()
	ctx1.writeError("errorMessage", "errorDebug")
	thread1.outStream.SetReadPos(17)
	assert(thread1.execSuccessful).IsTrue()
	assert(thread1.outStream.GetWritePos()).Equals(17)
	assert(thread1.outStream.GetReadPos()).Equals(17)
}

func TestRpcContext_Error(t *testing.T) {
	assert := NewAssert(t)

	// ctx is ok
	thread := newThread(nil)
	thread.stop()
	ctx := rpcContext{thread: unsafe.Pointer(thread)}
	assert(ctx.Error(nil)).IsNil()
	assert(thread.outStream.GetWritePos()).Equals(17)
	assert(thread.outStream.GetReadPos()).Equals(17)

	assert(
		ctx.Error(NewRPCErrorByDebug("errorMessage", "errorDebug")),
	).IsNil()
	thread.outStream.SetReadPos(17)
	assert(thread.outStream.ReadBool()).Equals(false, true)
	assert(thread.outStream.ReadString()).Equals("errorMessage", true)
	assert(thread.outStream.ReadString()).Equals("errorDebug", true)

	// ctx have execEchoNode
	thread1 := newThread(nil)
	thread1.stop()
	thread1.execEchoNode = &rpcEchoNode{debugString: "nodeDebug"}
	ctx1 := rpcContext{thread: unsafe.Pointer(thread1)}
	assert(
		ctx1.Error(NewRPCErrorByDebug("errorMessage", "errorDebug")),
	).IsNil()
	thread1.outStream.SetReadPos(17)
	assert(thread1.outStream.ReadBool()).Equals(false, true)
	assert(thread1.outStream.ReadString()).Equals("errorMessage", true)
	assert(thread1.outStream.ReadString()).Equals("errorDebug\nnodeDebug", true)
}

func TestRpcContext_Errorf(t *testing.T) {
	assert := NewAssert(t)

	// ctx is ok
	thread := newThread(nil)
	thread.stop()
	ctx := rpcContext{thread: unsafe.Pointer(thread)}
	assert(ctx.Errorf("error%s", "Message")).IsNil()
	thread.outStream.SetReadPos(17)
	assert(thread.outStream.ReadBool()).Equals(false, true)
	assert(thread.outStream.ReadString()).Equals("errorMessage", true)
	dbgMessage, ok := thread.outStream.ReadString()
	assert(ok).IsTrue()
	assert(dbgMessage).Contains("TestRpcContext_Errorf")
}
