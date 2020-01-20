package rpc

import (
	"os"
	"path"
	"runtime"
	"testing"
)

type TestFuncCache struct{}

func (p *TestFuncCache) Get(fnString string) FuncCacheType {
	switch fnString {
	case "S":
		return func(c *rpcContext, s *RPCStream, f interface{}) bool {
			h, ok := s.ReadString()
			if !ok || s.CanRead() {
				return false
			}
			f.(func(*rpcContext, string) *rpcReturn)(c, h)
			return true
		}
	default:
		return nil
	}
}

func TestFnCache_basic(t *testing.T) {
	assert := newAssert(t)
	_, file, _, _ := runtime.Caller(0)

	processor0 := newRPCProcessor(nil, 16, 32, nil, nil)
	assert(processor0.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-0.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-0.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-0.go")))

	processor1 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor1.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context) Return {
			return ctx.OK(true)
		}), "")
	assert(processor1.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-1.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-1.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-1.go")))

	processor2 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor2.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Bool) Return {
			return ctx.OK(true)
		}), "")
	assert(processor2.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-2.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-2.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-2.go")))

	processor3 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor3.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Int64) Return {
			return ctx.OK(true)
		}), "")
	assert(processor3.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-3.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-3.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-3.go")))

	processor4 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor4.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Uint64) Return {
			return ctx.OK(true)
		}), "")
	assert(processor4.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-4.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-4.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-4.go")))

	processor5 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor5.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Float64) Return {
			return ctx.OK(true)
		}), "")
	assert(processor5.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-5.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-5.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-5.go")))

	processor6 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor6.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ String) Return {
			return ctx.OK(true)
		}), "")
	assert(processor6.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-6.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-6.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-6.go")))

	processor7 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor7.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Bytes) Return {
			return ctx.OK(true)
		}), "")
	assert(processor7.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-7.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-7.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-7.go")))

	processor8 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor8.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Array) Return {
			return ctx.OK(true)
		}), "")
	assert(processor8.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-8.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-8.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-8.go")))

	processor9 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor9.AddService("abc", NewService().
		Echo("sayHello", true, func(ctx Context, _ Map) Return {
			return ctx.OK(true)
		}), "")
	assert(processor9.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-9.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-9.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-9.go")))

	processor10 := newRPCProcessor(nil, 16, 32, nil, nil)
	_ = processor10.AddService("abc", NewService().
		Echo("sayHello", true, func(
			ctx Context, _ Bool, _ Int64, _ Uint64, _ Float64, _ String,
			_ Bytes, _ Array, _ Map,
		) Return {
			return ctx.OK(true)
		}), "")
	assert(processor10.BuildCache(
		"pkgName",
		path.Join(path.Dir(file), "_tmp_/fncache-basic-10.go"),
	)).IsNil()
	assert(readStringFromFile(
		path.Join(path.Dir(file), "_snapshot_/fncache-basic-10.snapshot"),
	)).Equals(readStringFromFile(
		path.Join(path.Dir(file), "_tmp_/fncache-basic-10.go")))

	_ = os.RemoveAll(path.Join(path.Dir(file), "_tmp_"))
}
