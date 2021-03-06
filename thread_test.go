package rpc

import (
	"reflect"
	"testing"
	"time"
)

func TestNewThread(t *testing.T) {
	assert := newAssert(t)

	processor := newRPCProcessor(nil, 16, 16, nil, nil)
	threadPool := newThreadPool(processor)

	thread := newThread(threadPool)
	assert(thread).IsNotNil()
	assert(thread.threadPool).Equals(threadPool)
	assert(thread.isRunning).Equals(true)
	assert(thread.threadPool.processor).Equals(processor)
	assert(len(thread.ch)).Equals(0)
	assert(cap(thread.ch)).Equals(0)
	assert(thread.inStream).IsNil()
	assert(thread.outStream).IsNotNil()
	assert(thread.execDepth).Equals(uint64(0))
	assert(thread.execEchoNode).IsNil()
	assert(len(thread.execArgs)).Equals(0)
	assert(cap(thread.execArgs)).Equals(16)
	assert(thread.execSuccessful).IsFalse()
	assert(thread.from).Equals("")
	assert(len(thread.closeCH)).Equals(0)
	assert(cap(thread.closeCH)).Equals(0)
}

func TestRpcThread_stop(t *testing.T) {
	assert := newAssert(t)
	thread := newThread(nil)
	assert(thread.stop()).IsTrue()
	assert(thread.stop()).IsFalse()

	timeoutMessageCH := make(chan string, 10)
	logger := NewLogger()
	processor := newRPCProcessor(logger, 16, 16, nil, nil)
	_ = processor.AddService(
		"user",
		NewService().Echo("sayHello", true, func(ctx Context) Return {
			time.Sleep(99999999 * time.Second)
			return ctx.OK(true)
		}),
		"",
	)

	stream := newStream()
	stream.WriteString("$.user:sayHello")
	stream.WriteUint64(3)
	stream.WriteString("#")

	processor.Start()
	processor.PutStream(stream)
	logger.Subscribe().Error = func(msg string) {
		timeoutMessageCH <- msg
	}
	processor.Stop()
	assert(<-timeoutMessageCH).Contains("Error: rpc-thread-pool: internal error")
	assert(<-timeoutMessageCH).Contains("Error: rpc-thread: stop: timeout")
}

func runWithProcessor(
	handler interface{},
	getStream func(processor *rpcProcessor) *rpcStream,
	onTest func(in *rpcStream, out *rpcStream, success bool),
) {
	retStreamCH := make(chan *rpcStream)
	retSuccessCH := make(chan bool)
	processor := newRPCProcessor(
		nil,
		16,
		16,
		func(stream *rpcStream, success bool) {
			retStreamCH <- stream
			retSuccessCH <- success
		},
		&TestFuncCache{},
	)
	_ = processor.AddService(
		"user",
		NewService().Echo("sayHello", true, handler),
		"",
	)

	inStream := getStream(processor)
	processor.Start()
	processor.PutStream(inStream)
	retStream := <-retStreamCH
	retSuccess := <-retSuccessCH
	onTest(inStream, retStream, retSuccess)
	processor.Stop()
}

func TestRpcThread_eval(t *testing.T) {
	assert := newAssert(t)

	// test basic
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.SetServerCallbackID(345343535345343535)
			stream.SetRouterID(345343535)
			stream.SetMachineID(345343535)
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(true)
			// inStream is reset
			assert(in.GetHeader()).Equals(out.GetHeader())
			assert(in.GetReadPos()).Equals(17)
			assert(in.GetWritePos()).Equals(17)
			assert(out.ReadBool()).Equals(true, true)
			assert(out.Read()).Equals("hello world", true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// test read echo path error
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			// path format error
			stream.WriteBytes([]byte("$.user:sayHello"))
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(false)
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc data format error", true)
			assert(out.Read()).Equals("", true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// echo path is not mounted
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.system:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(false)
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).
				Equals("rpc-server: echo path $.system:sayHello is not mounted", true)
			assert(out.Read()).Equals("", true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// depth data format error
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			// depth type error
			stream.WriteInt64(3)
			stream.WriteString("#")
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(false)
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc data format error", true)
			assert(out.Read()).Equals("", true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// depth is overflow
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(17)
			stream.WriteString("#")
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(false)
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals(
				"rpc current call depth(17) is overflow. limited(16)",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
		},
	)

	// from data format error
	runWithProcessor(
		func(ctx Context, name string) Return {
			return ctx.OK("hello " + name)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteBool(true)
			stream.WriteString("world")
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(false)
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc data format error", true)
			assert(out.Read()).Equals("", true)
		},
	)

	// OK, call with all type value
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint64(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).Equals(true)
			assert(out.ReadBool()).Equals(true, true)
			assert(out.Read()).Equals(true, true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 1st param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(3)
			stream.Write(int64(3))
			stream.Write(uint64(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Int64, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 2nd param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(true)
			stream.Write(uint64(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Bool, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 3rd param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(true)
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Bool, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 4th param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint(3))
			stream.Write(true)
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Bool, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 5th param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint(3))
			stream.Write(float64(3))
			stream.Write(true)
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.Bool, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 6th param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(true)
			stream.Write(Array{1})
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bool, rpc.Array, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 7th param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(true)
			stream.Write(Map{"name": "world"})
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Bool, rpc.Map) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// error with 8th param
	runWithProcessor(
		func(ctx Context,
			b bool, i int64, u uint64, f float64, s string,
			x Bytes, a Array, m Map,
		) Return {
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			stream.Write(int64(3))
			stream.Write(uint(3))
			stream.Write(float64(3))
			stream.Write("hello")
			stream.Write(([]byte)("world"))
			stream.Write(Array{1})
			stream.Write(true)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Bool) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Int64, rpc.Uint64, "+
				"rpc.Float64, rpc.String, rpc.Bytes, rpc.Array, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// test nil rpcBytes
	runWithProcessor(
		func(ctx Context, a Bytes) Return {
			if a != nil {
				return ctx.Errorf("param is not nil")
			}

			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(nil)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsTrue()
			assert(out.ReadBool()).Equals(true, true)
			assert(out.Read()).Equals(true, true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// test nil rpcArray
	runWithProcessor(
		func(ctx Context, a Array) Return {
			if a != nil {
				return ctx.Errorf("param is not nil")
			}
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(nil)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsTrue()
			assert(out.ReadBool()).Equals(true, true)
			assert(out.Read()).Equals(true, true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// test nil rpcMap
	runWithProcessor(
		func(ctx Context, a Map) Return {
			if a != nil {
				return ctx.Errorf("param is not nil")
			}
			return ctx.OK(true)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(nil)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsTrue()
			assert(out.ReadBool()).Equals(true, true)
			assert(out.Read()).Equals(true, true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// test unsupported type
	runWithProcessor(
		func(ctx Context, a bool) Return {
			return ctx.OK(a)
		},
		func(processor *rpcProcessor) *rpcStream {
			echoNode := processor.echosMap["$.user:sayHello"]
			echoNode.argTypes[1] = reflect.ValueOf(int16(0)).Type()
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, rpc.Bool) rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// nil text
	runWithProcessor(
		func(ctx Context, bVal bool, rpcMap Map) Return {
			return ctx.OK(bVal)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(nil)
			stream.Write(nil)
			stream.Write(nil)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc echo arguments not match\n"+
				"Called: $.user:sayHello(rpc.Context, <nil>, rpc.Map, <nil>) "+
				"rpc.Return\n"+
				"Required: $.user:sayHello(rpc.Context, rpc.Bool, rpc.Map) rpc.Return",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(ok).IsTrue()
			assert(dbgMessage).Contains("$.user:sayHello")
			assert(out.CanRead()).IsFalse()
		},
	)

	// stream is broken
	runWithProcessor(
		func(ctx Context, bVal bool, rpcMap Map) Return {
			return ctx.OK(bVal)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write("helloWorld")
			stream.SetWritePos(stream.GetWritePos() - 1)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals("rpc data format error", true)
			assert(out.Read()).Equals("", true)
			assert(out.CanRead()).IsFalse()
		},
	)

	// call function error
	runWithProcessor(
		func(ctx Context, bVal bool) Return {
			if bVal {
				panic("this is a error")
			}
			return ctx.OK(bVal)
		},
		func(_ *rpcProcessor) *rpcStream {
			stream := newStream()
			stream.WriteString("$.user:sayHello")
			stream.WriteUint64(3)
			stream.WriteString("#")
			stream.Write(true)
			return stream
		},
		func(in *rpcStream, out *rpcStream, success bool) {
			assert(success).IsFalse()
			assert(out.ReadBool()).Equals(false, true)
			assert(out.Read()).Equals(
				"rpc-server: $.user:sayHello(rpc.Context, rpc.Bool) rpc.Return: "+
					"runtime error: this is a error",
				true,
			)
			dbgMessage, ok := out.Read()
			assert(dbgMessage).Contains("TestRpcThread_eval")
			assert(ok).IsTrue()
			assert(out.CanRead()).IsFalse()
		},
	)
}
