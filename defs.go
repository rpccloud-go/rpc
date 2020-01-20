package common

import (
	"reflect"
)

var (
	nilContext  = RPCContext(nil)
	nilReturn   = RPCReturn(nil)
	contextType = reflect.ValueOf(nilContext).Type()
	returnType  = reflect.ValueOf(nilReturn).Type()
	boolType    = reflect.ValueOf(true).Type()
	int64Type   = reflect.ValueOf(int64(0)).Type()
	uint64Type  = reflect.ValueOf(uint64(0)).Type()
	float64Type = reflect.ValueOf(float64(0)).Type()
	stringType  = reflect.ValueOf("").Type()
	bytesType   = reflect.ValueOf(RPCBytes{}).Type()
	arrayType   = reflect.ValueOf(RPCArray{}).Type()
	mapType     = reflect.ValueOf(RPCMap{}).Type()
)

// RPCCache ...
type RPCCache interface {
	Get(fnString string) RPCCacheFunc
}

// RPCCacheFunc ...
type RPCCacheFunc = func(
	ctx *rpcContext,
	stream *RPCStream,
	fn interface{},
) bool

type fnProcessorCallback = func(
	stream *RPCStream,
	success bool,
)

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

// RPCContext ...
type RPCContext = *rpcContext

// RPCBool ...
type RPCBool = bool

// RPCInt ...
type RPCInt = int64

// RPCUint ...
type RPCUint = uint64

// RPCFloat ...
type RPCFloat = float64

// RPCString ...
type RPCString = string

// RPCBytes ...
type RPCBytes = []byte

// RPCArray ...
type RPCArray = []interface{}

// RPCMap common Map type
type RPCMap = map[string]interface{}

// RPCAny ...
type RPCAny = interface{}

// rpcReturn is rpc function return type
type rpcReturn struct{}

// RPCReturn ...
type RPCReturn = *rpcReturn
