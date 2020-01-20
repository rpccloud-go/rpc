package rpc

import (
	"fmt"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

type rpcThread struct {
	threadPool     *rpcThreadPool
	isRunning      bool
	ch             chan *RPCStream
	inStream       *RPCStream
	outStream      *RPCStream
	execDepth      uint64
	execEchoNode   *rpcEchoNode
	execArgs       []reflect.Value
	execSuccessful bool
	from           string
	closeCH        chan bool
	AutoLock
}

func newThread(threadPool *rpcThreadPool) *rpcThread {
	ret := &rpcThread{
		threadPool:     threadPool,
		isRunning:      true,
		ch:             make(chan *RPCStream),
		inStream:       nil,
		outStream:      NewRPCStream(),
		execDepth:      0,
		execEchoNode:   nil,
		execArgs:       make([]reflect.Value, 0, 16),
		execSuccessful: false,
		from:           "",
		closeCH:        make(chan bool),
	}

	go func() {
		for stream := <-ret.ch; stream != nil; stream = <-ret.ch {
			ret.eval(stream)
		}
		ret.closeCH <- true
	}()

	return ret
}

func (p *rpcThread) stop() bool {
	return p.CallWithLock(func() interface{} {
		if p.isRunning {
			p.isRunning = false
			close(p.ch)
			select {
			case ret := <-p.closeCH:
				close(p.closeCH)
				return ret
			case <-time.After(6 * time.Second):
				if p.threadPool != nil &&
					p.threadPool.processor != nil &&
					p.threadPool.processor.logger != nil {
					p.threadPool.processor.logger.Error("rpc-thread: stop: timeout")
				}
				close(p.closeCH)
				return false
			}
		}
		return false
	}).(bool)
}

func (p *rpcThread) put(stream *RPCStream) {
	p.ch <- stream
}

func (p *rpcThread) eval(inStream *RPCStream) *rpcReturn {
	processor := p.threadPool.processor
	timeStart := TimeNowNS()
	// create context
	p.inStream = inStream
	p.execSuccessful = false
	ctx := &rpcContext{thread: unsafe.Pointer(p)}

	defer func() {
		if err := recover(); err != nil && p.execEchoNode != nil {
			ctx.writeError(
				fmt.Sprintf(
					"rpc-server: %s: runtime error: %s",
					p.execEchoNode.callString,
					err,
				),
				GetStackString(1),
			)
		}
		if p.execEchoNode != nil {
			p.execEchoNode.indicator.Count(
				time.Duration(TimeNowNS()-timeStart),
				p.from,
				p.execSuccessful,
			)
		}
		ctx.stop()
		inStream.Reset()
		retStream := p.outStream
		p.outStream = inStream
		p.from = ""
		p.execDepth = 0
		p.execEchoNode = nil
		p.execArgs = p.execArgs[:0]
		if processor.callback != nil {
			processor.callback(retStream, p.execSuccessful)
		}
		p.threadPool.freeThread(p)
	}()

	// copy head
	copy(p.outStream.GetHeader(), inStream.GetHeader())

	// read echo path
	echoPath, ok := inStream.ReadUnsafeString()
	if !ok {
		return ctx.writeError("rpc data format error", "")
	}
	if p.execEchoNode, ok = processor.echosMap[echoPath]; !ok {
		return ctx.writeError(
			fmt.Sprintf("rpc-server: echo path %s is not mounted", echoPath),
			"",
		)
	}

	// read depth
	if p.execDepth, ok = inStream.ReadUint64(); !ok {
		return ctx.writeError("rpc data format error", "")
	}
	if p.execDepth > processor.maxCallDepth {
		return ctx.Errorf(
			"rpc current call depth(%d) is overflow. limited(%d)",
			p.execDepth,
			processor.maxCallDepth,
		)
	}

	// read from
	if p.from, ok = inStream.ReadUnsafeString(); !ok {
		return ctx.writeError("rpc data format error", "")
	}

	// build callArgs
	argStartPos := inStream.GetReadPos()

	if fnCache := p.execEchoNode.cacheFN; fnCache != nil {
		ok = fnCache(ctx, inStream, p.execEchoNode.echoMeta.handler)
	} else {
		p.execArgs = append(p.execArgs, reflect.ValueOf(ctx))
		for i := 1; i < len(p.execEchoNode.argTypes); i++ {
			var rv reflect.Value

			switch p.execEchoNode.argTypes[i].Kind() {
			case reflect.Int64:
				if iVar, success := inStream.ReadInt64(); success {
					rv = reflect.ValueOf(iVar)
				} else {
					ok = false
				}
				break
			case reflect.Uint64:
				if uVar, success := inStream.ReadUint64(); success {
					rv = reflect.ValueOf(uVar)
				} else {
					ok = false
				}
				break
			case reflect.Float64:
				if fVar, success := inStream.ReadFloat64(); success {
					rv = reflect.ValueOf(fVar)
				} else {
					ok = false
				}
				break
			case reflect.Bool:
				if bVar, success := inStream.ReadBool(); success {
					rv = reflect.ValueOf(bVar)
				} else {
					ok = false
				}
				break
			case reflect.String:
				sVar, success := inStream.ReadString()
				if !success {
					ok = false
				} else {
					rv = reflect.ValueOf(sVar)
				}
				break
			default:
				switch p.execEchoNode.argTypes[i] {
				case bytesType:
					if xVar, success := inStream.ReadBytes(); success {
						rv = reflect.ValueOf(xVar)
					} else {
						ok = false
					}
					break
				case arrayType:
					if aVar, success := inStream.ReadArray(); success {
						rv = reflect.ValueOf(aVar)
					} else {
						ok = false
					}
					break
				case mapType:
					if mVar, success := inStream.ReadMap(); success {
						rv = reflect.ValueOf(mVar)
					} else {
						ok = false
					}
					break
				default:
					ok = false
				}
			}

			if !ok {
				break
			}

			p.execArgs = append(p.execArgs, rv)
		}

		if ok && !inStream.CanRead() {
			p.execEchoNode.reflectFn.Call(p.execArgs)
		}
	}

	// if stream data format error
	if !ok || inStream.CanRead() {
		inStream.SetReadPos(argStartPos)
		remoteArgsType := make([]string, 0, 0)
		remoteArgsType = append(remoteArgsType, convertTypeToString(contextType))
		for inStream.CanRead() {
			val, ok := inStream.Read()

			if !ok {
				return ctx.writeError("rpc data format error", "")
			}

			if val == nil {
				if len(remoteArgsType) < len(p.execEchoNode.argTypes) {
					argType := p.execEchoNode.argTypes[len(remoteArgsType)]

					if argType == bytesType ||
						argType == arrayType ||
						argType == mapType {
						remoteArgsType = append(
							remoteArgsType,
							convertTypeToString(argType),
						)
					} else {
						remoteArgsType = append(remoteArgsType, "<nil>")
					}
				} else {
					remoteArgsType = append(remoteArgsType, "<nil>")
				}
			} else {
				remoteArgsType = append(
					remoteArgsType,
					convertTypeToString(reflect.ValueOf(val).Type()),
				)
			}
		}

		return ctx.writeError(
			fmt.Sprintf(
				"rpc echo arguments not match\nCalled: %s(%s) %s\nRequired: %s",
				echoPath,
				strings.Join(remoteArgsType, ", "),
				convertTypeToString(returnType),
				p.execEchoNode.callString,
			),
			p.execEchoNode.debugString,
		)
	}

	return nilReturn
}
