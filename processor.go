package rpc

import (
	"fmt"
	"reflect"
	"regexp"
	"runtime"
	"strings"
)

const (
	rootName                 = "$"
	numOfThreadPerThreadPool = 2
	numOfThreadPoolPerCore   = 2
	numOfMinThreadPool       = 2
	numOfMaxThreadPool       = 2
	//numOfThreadPerThreadPool = 8192
	//numOfThreadPoolPerCore   = 2
	//numOfMinThreadPool       = 2
	//numOfMaxThreadPool       = 64
)

var (
	nodeNameRegex = regexp.MustCompile(`^[_0-9a-zA-Z]+$`)
	echoNameRegex = regexp.MustCompile(`^[_a-zA-Z][_0-9a-zA-Z]*$`)
)

type rpcEchoNode struct {
	serviceNode *rpcServiceNode
	path        string
	echoMeta    *rpcEchoMeta
	cacheFN     FuncCacheType
	reflectFn   reflect.Value
	callString  string
	debugString string
	argTypes    []reflect.Type
	indicator   *rpcPerformanceIndicator
}

type rpcServiceNode struct {
	path    string
	addMeta *rpcNodeMeta
	depth   uint
}

// rpcProcessor ...
type rpcProcessor struct {
	isRunning    bool
	logger       *Logger
	fnCache      FuncCache
	callback     fnProcessorCallback
	echosMap     map[string]*rpcEchoNode
	nodesMap     map[string]*rpcServiceNode
	threadPools  []*rpcThreadPool
	maxNodeDepth uint64
	maxCallDepth uint64
	rpcAutoLock
}

var fnGetRuntimeNumberOfCPU = func() int {
	return runtime.NumCPU()
}

// newRPCProcessor ...
func newRPCProcessor(
	logger *Logger,
	maxNodeDepth uint,
	maxCallDepth uint,
	callback fnProcessorCallback,
	fnCache FuncCache,
) *rpcProcessor {
	numOfThreadPool := uint32(fnGetRuntimeNumberOfCPU() * numOfThreadPoolPerCore)
	if numOfThreadPool < numOfMinThreadPool {
		numOfThreadPool = numOfMinThreadPool
	}
	if numOfThreadPool > numOfMaxThreadPool {
		numOfThreadPool = numOfMaxThreadPool
	}

	ret := &rpcProcessor{
		isRunning:    false,
		logger:       logger,
		fnCache:      fnCache,
		callback:     callback,
		echosMap:     make(map[string]*rpcEchoNode),
		nodesMap:     make(map[string]*rpcServiceNode),
		threadPools:  make([]*rpcThreadPool, numOfThreadPool, numOfThreadPool),
		maxNodeDepth: uint64(maxNodeDepth),
		maxCallDepth: uint64(maxCallDepth),
	}

	// mount root node
	ret.nodesMap[rootName] = &rpcServiceNode{
		path:    rootName,
		addMeta: nil,
		depth:   0,
	}

	return ret
}

// Start ...
func (p *rpcProcessor) Start() bool {
	return p.CallWithLock(func() interface{} {
		if !p.isRunning {
			p.isRunning = true
			for i := 0; i < len(p.threadPools); i++ {
				p.threadPools[i] = newThreadPool(p)
			}
			return true
		}

		return false
	}).(bool)
}

// Stop ...
func (p *rpcProcessor) Stop() bool {
	return p.CallWithLock(func() interface{} {
		if p.isRunning {
			for i := 0; i < len(p.threadPools); i++ {
				p.threadPools[i].stop()
				p.threadPools[i] = nil
			}
			p.isRunning = false
			return true
		}

		return false
	}).(bool)
}

// PutStream ...
func (p *rpcProcessor) PutStream(stream *rpcStream) bool {
	// PutStream stream in a random thread pool
	threadPool := p.threadPools[int(getRandUint32())%len(p.threadPools)]
	if threadPool != nil {
		if thread := threadPool.allocThread(); thread != nil {
			thread.put(stream)
			return true
		}
		return false
	}

	return false
}

// BuildCache ...
func (p *rpcProcessor) BuildCache(pkgName string, path string) error {
	retMap := make(map[string]bool)
	for _, echo := range p.echosMap {
		if fnTypeString, ok := getFuncKind(echo.echoMeta.handler); ok {
			retMap[fnTypeString] = true
		}
	}

	fnKinds := make([]string, 0)
	for key := range retMap {
		fnKinds = append(fnKinds, key)
	}

	return buildFuncCache(pkgName, path, fnKinds)
}

// AddService ...
func (p *rpcProcessor) AddService(
	name string,
	service Service,
	debug string,
) Error {
	serviceMeta, ok := service.(*rpcService)
	if !ok {
		return NewErrorByDebug(
			"Service is nil",
			debug,
		)
	}

	return p.mountNode(rootName, &rpcNodeMeta{
		name:        name,
		serviceMeta: serviceMeta,
		debug:       debug,
	})
}

func (p *rpcProcessor) mountNode(
	parentServiceNodePath string,
	nodeMeta *rpcNodeMeta,
) Error {
	// check nodeMeta is not nil
	if nodeMeta == nil {
		return NewError("rpc: mountNode: nodeMeta is nil")
	}

	// check nodeMeta.name is valid
	if !nodeNameRegex.MatchString(nodeMeta.name) {
		return NewErrorByDebug(
			fmt.Sprintf("Service name \"%s\" is illegal", nodeMeta.name),
			nodeMeta.debug,
		)
	}

	// check nodeMeta.serviceMeta is not nil
	if nodeMeta.serviceMeta == nil {
		return NewErrorByDebug(
			"Service is nil",
			nodeMeta.debug,
		)
	}

	// check max node depth overflow
	parentNode, ok := p.nodesMap[parentServiceNodePath]
	if !ok {
		return NewErrorByDebug(
			"rpc: mountNode: parentNode is nil",
			nodeMeta.debug,
		)
	}
	servicePath := parentServiceNodePath + "." + nodeMeta.name
	if uint64(parentNode.depth+1) > p.maxNodeDepth {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Service path depth %s is too long, it must be less or equal than %d",
				servicePath,
				p.maxNodeDepth,
			),
			nodeMeta.debug,
		)
	}

	// check the mount path is not occupied
	if item, ok := p.nodesMap[servicePath]; ok {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Service name \"%s\" is duplicated",
				nodeMeta.name,
			),
			fmt.Sprintf(
				"Current:\n%s\nConflict:\n%s",
				addPrefixPerLine(nodeMeta.debug, "\t"),
				addPrefixPerLine(item.addMeta.debug, "\t"),
			),
		)
	}

	node := &rpcServiceNode{
		path:    servicePath,
		addMeta: nodeMeta,
		depth:   parentNode.depth + 1,
	}

	// mount the node
	p.nodesMap[servicePath] = node

	// mount the echos
	for _, echoMeta := range nodeMeta.serviceMeta.echos {
		err := p.mountEcho(node, echoMeta)
		if err != nil {
			delete(p.nodesMap, servicePath)
			return err
		}
	}

	// mount children
	for _, v := range nodeMeta.serviceMeta.children {
		err := p.mountNode(node.path, v)
		if err != nil {
			delete(p.nodesMap, servicePath)
			return err
		}
	}

	return nil
}

func (p *rpcProcessor) mountEcho(
	serviceNode *rpcServiceNode,
	echoMeta *rpcEchoMeta,
) Error {
	// check the node is nil
	if serviceNode == nil {
		return NewError("rpc: mountEcho: node is nil")
	}

	// check the echoMeta is nil
	if echoMeta == nil {
		return NewError("rpc: mountEcho: echoMeta is nil")
	}

	// check the name
	if !echoNameRegex.MatchString(echoMeta.name) {
		return NewErrorByDebug(
			fmt.Sprintf("Echo name %s is illegal", echoMeta.name),
			echoMeta.debug,
		)
	}

	// check the echo path is not occupied
	echoPath := serviceNode.path + ":" + echoMeta.name
	if item, ok := p.echosMap[echoPath]; ok {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Echo name %s is duplicated",
				echoMeta.name,
			),
			fmt.Sprintf(
				"Current:\n%s\nConflict:\n%s",
				addPrefixPerLine(echoMeta.debug, "\t"),
				addPrefixPerLine(item.echoMeta.debug, "\t"),
			),
		)
	}

	// check the echo handler is nil
	if echoMeta.handler == nil {
		return NewErrorByDebug(
			"Echo handler is nil",
			echoMeta.debug,
		)
	}

	// Check echo handler is Func
	fn := reflect.ValueOf(echoMeta.handler)
	if fn.Kind() != reflect.Func {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Echo handler must be func(ctx %s, ...) %s",
				convertTypeToString(contextType),
				convertTypeToString(returnType),
			),
			echoMeta.debug,
		)
	}

	// Check echo handler arguments types
	argumentsErrorPos := getArgumentsErrorPosition(fn)
	if argumentsErrorPos == 0 {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Echo handler 1st argument type must be %s",
				convertTypeToString(contextType),
			),
			echoMeta.debug,
		)
	} else if argumentsErrorPos > 0 {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Echo handler %s argument type <%s> not supported",
				convertOrdinalToString(1+uint(argumentsErrorPos)),
				fmt.Sprintf("%s", fn.Type().In(argumentsErrorPos)),
			),
			echoMeta.debug,
		)
	}

	// Check return type
	if fn.Type().NumOut() != 1 ||
		fn.Type().Out(0) != reflect.ValueOf(nilReturn).Type() {
		return NewErrorByDebug(
			fmt.Sprintf(
				"Echo handler return type must be %s",
				convertTypeToString(returnType),
			),
			echoMeta.debug,
		)
	}

	// mount the echoRecord
	fileLine := ""
	debugArr := findLinesByPrefix(echoMeta.debug, "-01")
	if len(debugArr) > 0 {
		arr := strings.Split(debugArr[0], " ")
		if len(arr) == 3 {
			fileLine = arr[2]
		}
	}

	argTypes := make([]reflect.Type, fn.Type().NumIn(), fn.Type().NumIn())
	argStrings := make([]string, fn.Type().NumIn(), fn.Type().NumIn())
	for i := 0; i < len(argTypes); i++ {
		argTypes[i] = fn.Type().In(i)
		argStrings[i] = convertTypeToString(argTypes[i])
	}
	argString := strings.Join(argStrings, ", ")

	cacheFN := FuncCacheType(nil)
	if fnTypeString, ok := getFuncKind(echoMeta.handler); ok && p.fnCache != nil {
		cacheFN = p.fnCache.Get(fnTypeString)
	}

	p.echosMap[echoPath] = &rpcEchoNode{
		serviceNode: serviceNode,
		path:        echoPath,
		echoMeta:    echoMeta,
		cacheFN:     cacheFN,
		reflectFn:   fn,
		callString: fmt.Sprintf(
			"%s(%s) %s",
			echoPath,
			argString,
			convertTypeToString(returnType),
		),
		debugString: fmt.Sprintf("%s %s", echoPath, fileLine),
		argTypes:    argTypes,
		indicator:   newPerformanceIndicator(),
	}

	if p.logger != nil {
		p.logger.Infof(
			"rpc: mounted %s %s",
			p.echosMap[echoPath].callString,
			fileLine,
		)
	}

	return nil
}
