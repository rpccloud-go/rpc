package rpc

import (
	"fmt"
	"github.com/gorilla/websocket"
	"math"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"
)

const (
	wsServerOpening    = int32(1)
	wsServerOpened     = int32(2)
	wsServerClosing    = int32(3)
	wsServerDidClosing = int32(4)
	wsServerClosed     = int32(5)
)

var (
	wsUpgradeManager = websocket.Upgrader{
		ReadBufferSize:    1024,
		WriteBufferSize:   1024,
		EnableCompression: true,
	}
)

type wsServerConn struct {
	id         uint32
	wsConn     unsafe.Pointer
	connIndex  uint32
	security   string
	deadlineNS int64
	streamCH   chan *rpcStream
	sequence   uint32
	sync.Mutex
}

func (p *wsServerConn) getSequence() uint32 {
	ret := uint32(0)
	p.Lock()
	ret = p.sequence
	p.Unlock()
	return ret
}

func (p *wsServerConn) setSequence(from uint32, to uint32) bool {
	ret := false
	p.Lock()
	if p.sequence == from && from != to {
		p.sequence = to
		ret = true
	}
	p.Unlock()
	return ret
}

// WebSocketServer is implement of INetServer via web socket
type WebSocketServer struct {
	processor     *rpcProcessor
	logger        *Logger
	status        int32
	readSizeLimit uint64
	readTimeoutNS uint64
	httpServer    *http.Server
	seed          uint32
	sync.Map
	sync.Mutex
}

// NewWebSocketServer create a WebSocketClient
func NewWebSocketServer(fnCache FuncCache) *WebSocketServer {
	server := &WebSocketServer{
		processor:     nil,
		logger:        NewLogger(),
		status:        wsServerClosed,
		readSizeLimit: 64 * 1024,
		readTimeoutNS: 60 * uint64(time.Second),
		httpServer:    nil,
		seed:          1,
	}

	server.processor = newRPCProcessor(
		server.logger,
		32,
		32,
		func(stream *rpcStream, success bool) {
			// Todo: deal error (chan close)
			if serverConn := server.getConnByID(
				stream.GetClientConnID(),
			); serverConn != nil {
				serverConn.streamCH <- stream
			}
		},
		fnCache,
	)
	return server
}

func (p *WebSocketServer) serverConnWriteRoutine(serverConn *wsServerConn) {
	ch := serverConn.streamCH
	for stream := <-ch; stream != nil; stream = <-ch {
		stream.SetClientConnID(0)
		for serverConn.security != "" {
			if wsConn := atomic.LoadPointer(&serverConn.wsConn); wsConn != nil {
				if err := (*websocket.Conn)(wsConn).WriteMessage(
					websocket.BinaryMessage,
					stream.GetBufferUnsafe(),
				); err == nil {
					// release old stream
					stream.Release()
					break
				} else {
					p.onError(serverConn, err.Error())
				}
			}
			time.Sleep(200 * time.Millisecond)
		}
	}
}

func (p *WebSocketServer) registerConn(
	wsConn *websocket.Conn,
	id uint32,
	security string,
) *wsServerConn {
	// id and security is ok
	if v, ok := p.Load(id); ok {
		serverConn, ok := v.(*wsServerConn)
		if ok && serverConn != nil && serverConn.security == security {
			atomic.StorePointer(&serverConn.wsConn, unsafe.Pointer(wsConn))
			atomic.StoreInt64(&serverConn.deadlineNS, 0)
			return serverConn
		}
	}

	p.Lock()
	ret := (*wsServerConn)(nil)
	for {
		p.seed++
		if p.seed == math.MaxUint32 {
			p.seed = 1
		}

		id = p.seed
		if _, ok := p.Load(id); !ok {
			ret = &wsServerConn{
				id:         id,
				sequence:   1,
				security:   GetRandString(32),
				wsConn:     unsafe.Pointer(wsConn),
				connIndex:  0,
				deadlineNS: 0,
				streamCH:   make(chan *rpcStream, 256),
			}
			p.Store(id, ret)
			go p.serverConnWriteRoutine(ret)
			break
		}
	}
	p.Unlock()
	return ret
}

func (p *WebSocketServer) unregisterConn(id uint32, force bool) bool {
	if serverConn, ok := p.Load(id); ok {
		if force {
			serverConn.(*wsServerConn).security = ""
		}
		atomic.StorePointer(&serverConn.(*wsServerConn).wsConn, nil)
		atomic.StoreInt64(
			&serverConn.(*wsServerConn).deadlineNS,
			timeNowNS()+35*int64(time.Second),
		)
		return true
	}

	return false
}

func (p *WebSocketServer) swipeConn() {
	for atomic.LoadInt32(&p.status) != wsServerClosed {
		nowNS := timeNowNS()
		p.Range(func(key, value interface{}) bool {
			v, ok := value.(*wsServerConn)
			if ok && v != nil {
				deadlineNS := atomic.LoadInt64(&v.deadlineNS)
				if deadlineNS > 0 && deadlineNS < nowNS {
					p.Delete(key)
					atomic.StorePointer(&v.wsConn, nil)
					v.security = ""
					close(v.streamCH)
					v.streamCH = nil
				}
			}
			return true
		})

		time.Sleep(500 * time.Millisecond)
	}
}

func (p *WebSocketServer) getConnByID(id uint32) *wsServerConn {
	if v, ok := p.Load(id); ok {
		return v.(*wsServerConn)
	}
	return nil
}

// AddService ...
func (p *WebSocketServer) AddService(
	name string,
	service Service,
) *WebSocketServer {
	err := p.processor.AddService(name, service, getStackString(1))
	if err != nil {
		p.logger.Error(err.Error())
	}
	return p
}

// StartBackground ...
func (p *WebSocketServer) StartBackground(
	host string,
	port uint16,
	path string,
) {
	wait := true
	go func() {
		err := p.Start(host, port, path)
		if err != nil {
			p.logger.Error(err.Error())
		}
		wait = false
	}()

	for wait {
		if atomic.LoadInt32(&p.status) == wsServerOpened {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// BuildFuncCache ...
func (p *WebSocketServer) BuildFuncCache(
	pkgName string,
	relativePath string,
) error {
	_, file, _, _ := runtime.Caller(1)
	return p.processor.BuildCache(
		pkgName,
		path.Join(path.Dir(file), relativePath),
	)
}

// Start make the WebSocketServer start serve
func (p *WebSocketServer) Start(
	host string,
	port uint16,
	path string,
) (ret Error) {
	if atomic.CompareAndSwapInt32(&p.status, wsServerClosed, wsServerOpening) {
		p.logger.Infof(
			"WebSocketServer: start at %s",
			getURLBySchemeHostPortAndPath("ws", host, port, path),
		)
		p.processor.Start()
		serverMux := http.NewServeMux()
		serverMux.HandleFunc(path, func(w http.ResponseWriter, req *http.Request) {
			if req != nil && req.Header != nil {
				req.Header.Del("Origin")
			}

			connID := uint32(0)
			connSecurity := ""
			keys, ok := req.URL.Query()["conn"]
			if ok && len(keys) == 1 {
				arr := strings.Split(keys[0], "-")
				if len(arr) == 2 {
					if parseID, err := strconv.ParseUint(arr[0], 10, 64); err == nil {
						connID = uint32(parseID)
						connSecurity = arr[1]
					}
				}
			}

			wsConn, err := wsUpgradeManager.Upgrade(w, req, nil)
			if err != nil {
				p.logger.Errorf("WebSocketServer: %s", err.Error())
				return
			}

			serverConn := p.registerConn(wsConn, connID, connSecurity)

			// set conn information
			connStream := newStream()
			connStream.SetClientCallbackID(0)
			connStream.WriteString("#.connection.openInformation")
			connStream.WriteUint64(uint64(serverConn.id))
			connStream.WriteString(serverConn.security)
			connStream.WriteUint64(uint64(serverConn.getSequence()))
			serverConn.streamCH <- connStream

			wsConn.SetReadLimit(int64(atomic.LoadUint64(&p.readSizeLimit)))
			p.onOpen(serverConn)
			defer func() {
				p.unregisterConn(serverConn.id, false)
				err := wsConn.Close()
				if err != nil {
					p.onError(serverConn, err.Error())
				}
				p.onClose(serverConn)
			}()

			for {
				nextTimeoutNS := timeNowNS() +
					int64(atomic.LoadUint64(&p.readTimeoutNS))
				if err := wsConn.SetReadDeadline(time.Unix(
					nextTimeoutNS/int64(time.Second),
					nextTimeoutNS%int64(time.Second),
				)); err != nil {
					p.onError(serverConn, err.Error())
					return
				}

				mt, message, err := wsConn.ReadMessage()
				if err != nil {
					if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
						p.onError(serverConn, err.Error())
					}
					return
				}
				switch mt {
				case websocket.BinaryMessage:
					stream := newStream()
					stream.SetWritePos(0)
					stream.PutBytes(message)

					connSequence := stream.GetClientSequence()
					callbackID := stream.GetClientCallbackID()

					// this is system instructions
					if callbackID == 0 {
						// ignore system instructions
						p.onError(serverConn, "unknown system instruction")
						return
					}

					if connSequence > 4000000000 {
						p.unregisterConn(serverConn.id, true)
					}

					// this is rpc callback function
					if serverConn.setSequence(connSequence, callbackID) {
						stream.SetClientConnID(serverConn.id)
						p.onStream(serverConn, stream)
					} else {
						stream.Release()
						p.onError(serverConn, "server sequence error")
						return
					}
				default:
					p.onError(serverConn, "unknown message type")
					return
				}
			}
		})

		p.Lock()
		p.httpServer = &http.Server{
			Addr:    fmt.Sprintf("%s:%d", host, port),
			Handler: serverMux,
		}
		p.Unlock()

		time.AfterFunc(250*time.Millisecond, func() {
			atomic.CompareAndSwapInt32(&p.status, wsServerOpening, wsServerOpened)
		})

		go p.swipeConn()

		ret := p.httpServer.ListenAndServe()
		time.Sleep(400 * time.Millisecond)

		p.Lock()
		p.httpServer = nil
		p.Unlock()

		p.processor.Stop()

		p.logger.Infof(
			"WebSocketServer: stopped",
		)

		if atomic.LoadInt32(&p.status) == wsServerClosing {
			atomic.StoreInt32(&p.status, wsServerDidClosing)
		} else {
			atomic.StoreInt32(&p.status, wsServerClosed)
		}

		if ret != nil && ret.Error() == "http: Server closed" {
			return nil
		}

		return NewErrorBySystemError(ret)
	}

	return NewError("WebSocketServer: has already been started")
}

// Close make the WebSocketServer stop serve
func (p *WebSocketServer) Close() Error {
	if atomic.CompareAndSwapInt32(&p.status, wsServerOpened, wsServerClosing) {
		err := NewErrorBySystemError(p.httpServer.Close())
		for !atomic.CompareAndSwapInt32(
			&p.status,
			wsServerDidClosing,
			wsServerClosed,
		) {
			time.Sleep(20 * time.Millisecond)
		}
		return err
	}
	return NewError(
		"WebSocketServer: close error, it is not opened",
	)
}

func (p *WebSocketServer) onOpen(serverConn *wsServerConn) {
	p.logger.Infof("WebSocketServerConn[%d]: opened", serverConn.id)
}

func (p *WebSocketServer) onError(serverConn *wsServerConn, msg string) {
	p.logger.Warnf("WebSocketServerConn[%d]: %s", serverConn.id, msg)
}

func (p *WebSocketServer) onClose(serverConn *wsServerConn) {
	p.logger.Infof("WebSocketServerConn[%d]: closed", serverConn.id)
}

func (p *WebSocketServer) onStream(_ *wsServerConn, stream *rpcStream) {
	p.processor.PutStream(stream)
}

// GetLogger get WebSocketServer logger
func (p *WebSocketServer) GetLogger() *Logger {
	return p.logger
}

// SetReadSizeLimit set WebSocketServer read limit in byte
func (p *WebSocketServer) SetReadSizeLimit(readLimit uint64) {
	atomic.StoreUint64(&p.readSizeLimit, readLimit)
}

// SetReadTimeoutMS set WebSocketServer timeout in millisecond
func (p *WebSocketServer) SetReadTimeoutMS(readTimeoutMS uint64) {
	atomic.StoreUint64(&p.readTimeoutNS, readTimeoutMS*uint64(time.Millisecond))
}
