package rpc

import (
	"reflect"
)

var (
	nilContext  = Context(nil)
	nilReturn   = Return(nil)
	contextType = reflect.ValueOf(nilContext).Type()
	returnType  = reflect.ValueOf(nilReturn).Type()
	boolType    = reflect.ValueOf(true).Type()
	int64Type   = reflect.ValueOf(int64(0)).Type()
	uint64Type  = reflect.ValueOf(uint64(0)).Type()
	float64Type = reflect.ValueOf(float64(0)).Type()
	stringType  = reflect.ValueOf("").Type()
	bytesType   = reflect.ValueOf(Bytes{}).Type()
	arrayType   = reflect.ValueOf(Array{}).Type()
	mapType     = reflect.ValueOf(Map{}).Type()
)

// FuncCache ...
type FuncCache interface {
	Get(fnString string) FuncCacheType
}

// FuncCacheType ...
type FuncCacheType = func(
	ctx *rpcContext,
	stream *RPCStream,
	fn interface{},
) bool

type fnProcessorCallback = func(
	stream *RPCStream,
	success bool,
)

// Error ...
type Error interface {
	GetMessage() string
	GetDebug() string
	AddDebug(debug string)
	Error() string
}

// Service ...
type Service interface {
	Echo(
		name string,
		export bool,
		handler interface{},
	) Service

	AddService(
		name string,
		service Service,
	) Service
}

// Context ...
type Context = *rpcContext

// Bool ...
type Bool = bool

// Int64 ...
type Int64 = int64

// Uint64 ...
type Uint64 = uint64

// Float64 ...
type Float64 = float64

// String ...
type String = string

// Bytes ...
type Bytes = []byte

// Array ...
type Array = []interface{}

// Map common Map type
type Map = map[string]interface{}

// Any ...
type Any = interface{}

// rpcReturn is rpc function return type
type rpcReturn struct{}

// Return ...
type Return = *rpcReturn
