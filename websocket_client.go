package rpc

import (
	"encoding/binary"
	"errors"
	"math"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// The connection is connecting
	wsClientRunning = int32(1)
	wsClientClosed  = int32(2)
)

type websocketClientCallback struct {
	id        uint32
	timeNS    int64
	ch        chan bool
	stream    *RPCStream
	isTimeout bool
}

// WebSocketClient is implement of INetClient via web socket
type WebSocketClient struct {
	//logger        *Logger
	status        int32
	conn          *websocket.Conn
	seed          uint32
	serverConn    string
	msgTimeoutNS  int64
	readTimeoutNS int64
	readSizeLimit int64
	urlString     string
	sendChannel   chan *websocketClientCallback
	sendCurrent   *websocketClientCallback
	doConnectCH   chan bool
	doSendCH      chan bool
	doTimeoutCH   chan bool
	sync.Map
	sync.Mutex
}

// NewWebSocketClient create a WebSocketClient, and connect to url
func NewWebSocketClient(urlString string) *WebSocketClient {
	client := &WebSocketClient{
		//logger:        NewLogger(),
		status:        wsClientRunning,
		conn:          nil,
		seed:          1,
		serverConn:    "324#asjdfie",
		msgTimeoutNS:  20 * int64(time.Second),
		readTimeoutNS: 60 * int64(time.Second),
		readSizeLimit: 10 * 1024 * 1024,
		urlString:     urlString,
		sendChannel:   make(chan *websocketClientCallback, 1024),
		sendCurrent:   nil,
		doConnectCH:   make(chan bool, 1),
		doSendCH:      make(chan bool, 1),
		doTimeoutCH:   make(chan bool, 1),
	}

	go client.doConnect()
	go client.doSend()
	go client.doTimeout()

	return client
}

func (p *WebSocketClient) isRunning() bool {
	return atomic.LoadInt32(&p.status) == wsClientRunning
}

func (p *WebSocketClient) doConnect() {
	time.Sleep(30 * time.Millisecond)
	for p.isRunning() {
		startConnMS := TimeNowMS()
		p.connect()
		connMS := TimeNowMS() - startConnMS
		if connMS < 2000 && p.isRunning() {
			time.Sleep(time.Duration(2000-connMS) * time.Millisecond)
		}
	}
	p.doConnectCH <- true
}

func (p *WebSocketClient) doSend() {
	for p.isRunning() {
		if p.sendCurrent == nil {
			p.sendCurrent = <-p.sendChannel
		}

		// ignore timeout msg
		if p.sendCurrent != nil && p.sendCurrent.isTimeout {
			continue
		}

		// try to send, if success, the program will continue soon
		if conn := p.getConn(); conn != nil && p.sendCurrent != nil {
			buf := p.sendCurrent.stream.GetBufferUnsafe()
			err := conn.WriteMessage(websocket.BinaryMessage, buf)
			if err == nil {
				p.sendCurrent = nil
				continue
			}
			p.onError(err.Error())
		}

		// wait if send error
		if p.isRunning() {
			time.Sleep(100 * time.Millisecond)
		}
	}
	p.doSendCH <- true
}

func (p *WebSocketClient) doTimeout() {
	for p.isRunning() {
		nowNS := TimeNowNS()
		p.Range(func(key, value interface{}) bool {
			v, ok := value.(*websocketClientCallback)
			if ok && v != nil {
				if nowNS-v.timeNS > p.msgTimeoutNS {
					v.isTimeout = true
					v.ch <- false
				}
			}
			return true
		})
		time.Sleep(100 * time.Millisecond)
	}
	p.doTimeoutCH <- true
}

func (p *WebSocketClient) readBinaryMessage(
	readTimeoutNS int64,
) ([]byte, bool, error) {
	if p.conn == nil {
		return nil, false, errors.New("connection is nil")
	}

	// set next read dead line
	nextTimeoutNS := TimeNowNS() + readTimeoutNS
	if err := p.conn.SetReadDeadline(time.Unix(
		nextTimeoutNS/int64(time.Second),
		nextTimeoutNS%int64(time.Second),
	)); err != nil {
		return nil, true, err
	}

	// read message
	mt, message, err := p.conn.ReadMessage()
	if err != nil {
		if !websocket.IsCloseError(err, websocket.CloseNormalClosure) {
			return nil, true, err
		}
		return nil, true, nil
	}
	switch mt {
	case websocket.BinaryMessage:
		return message, false, nil
	case websocket.CloseMessage:
		return nil, true, nil
	default:
		return nil, true, errors.New("unknown message type")
	}
}

func (p *WebSocketClient) setConn(conn *websocket.Conn) {
	p.Lock()
	p.conn = conn
	p.Unlock()
}

func (p *WebSocketClient) getConn() *websocket.Conn {
	p.Lock()
	ret := p.conn
	p.Unlock()
	return ret
}

func (p *WebSocketClient) connect() {
	if p.getConn() == nil {
		// parse URL
		requestURL, err := url.Parse(p.urlString)
		if err != nil {
			p.onError(err.Error())
			return
		}
		query := requestURL.Query()
		query.Add("conn", p.serverConn)
		requestURL.RawQuery = query.Encode()

		// Dial
		if conn, _, err := websocket.DefaultDialer.Dial(
			requestURL.String(),
			nil,
		); err == nil {
			p.setConn(conn)
		} else {
			p.onError(err.Error())
			p.setConn(nil)
			return
		}

		// set read size limit
		p.conn.SetReadLimit(p.readSizeLimit)

		// receive server conn info
		msg, needClose, err := p.readBinaryMessage(2 * int64(time.Second))
		if err != nil {
			p.onError(err.Error())
		}
		if msg != nil {
			p.serverConn = string(msg)
		}
		if needClose || p.serverConn == "" {
			if err := p.conn.Close(); err != nil {
				p.onError(err.Error())
			}
			p.setConn(nil)
			return
		}

		p.onOpen()

		for {
			msg, needClose, err := p.readBinaryMessage(p.readTimeoutNS)
			if err != nil {
				p.onError(err.Error())
			}
			if needClose {
				break
			}
			p.onBinary(msg)
		}

		if err := p.conn.Close(); err != nil {
			p.onError(err.Error())
		}

		p.onClose()
		p.setConn(nil)
	}
}

func (p *WebSocketClient) registerCallback() *websocketClientCallback {
	ret := (*websocketClientCallback)(nil)
	p.Lock()
	for {
		p.seed++
		if p.seed == math.MaxUint32 {
			p.seed = 1
		}
		if _, ok := p.Load(p.seed); !ok {
			ret = &websocketClientCallback{
				id:        p.seed,
				timeNS:    TimeNowNS(),
				ch:        make(chan bool),
				stream:    newStream(),
				isTimeout: false,
			}
			p.Store(ret.id, ret)
			break
		}
	}
	p.Unlock()
	return ret
}

func (p *WebSocketClient) unregisterCallback(key uint32) bool {
	if _, ok := p.Load(key); ok {
		p.Delete(key)
		return true
	}
	return false
}

func (p *WebSocketClient) getCallbackByID(key uint32) *websocketClientCallback {
	if v, ok := p.Load(key); ok {
		return v.(*websocketClientCallback)
	}
	return nil
}

// SendMessage ...
func (p *WebSocketClient) SendMessage(
	target string,
	args ...interface{},
) (interface{}, Error) {
	if !p.isRunning() {
		return nil, NewError("client closed")
	}

	callback := p.registerCallback()
	defer p.unregisterCallback(callback.id)

	stream := callback.stream
	// set client callback id
	stream.SetClientCallbackID(callback.id)
	// write target
	stream.WriteString(target)
	// write depth
	stream.WriteUint64(0)
	// write from
	stream.WriteString("@")

	for i := 0; i < len(args); i++ {
		if stream.Write(args[i]) != rpcStreamWriteOK {
			return nil, NewError("args not supported")
		}
	}

	// send to channel
	p.sendChannel <- callback

	if response := <-callback.ch; !response {
		return nil, NewError("timeout")
	}

	success, ok := stream.ReadBool()
	if !ok {
		return nil, NewError("data format error")
	}

	if !success {
		message, ok := stream.ReadString()
		if !ok {
			return nil, NewError("data format error")
		}
		debug, ok := stream.ReadString()
		if !ok {
			return nil, NewError("data format error")
		}
		return nil, NewErrorByDebug(message, debug)
	}

	if ret, ok := stream.Read(); ok {
		return ret, nil
	}
	return nil, NewError("data format error")
}

// Close close the WebSocketClient
func (p *WebSocketClient) Close() (ret Error) {
	if atomic.CompareAndSwapInt32(&p.status, wsClientRunning, wsClientClosed) {
		close(p.sendChannel)
		if conn := p.getConn(); conn != nil {
			if err := p.conn.WriteMessage(
				websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			); err != nil {
				p.onError(err.Error())
				ret = NewErrorBySystemError(err)
			}
		}
		<-p.doConnectCH
		<-p.doSendCH
		<-p.doTimeoutCH
	} else {
		ret = NewError("WebSocketClient: client is not running")
	}
	return
}

func (p *WebSocketClient) onOpen() {
	//fmt.Println("client onOpen", conn)
}

func (p *WebSocketClient) onError(msg string) {
	//fmt.Println("client onError", conn, err)
}

func (p *WebSocketClient) onClose() {
	//fmt.Println("client onClose", conn)
}

func (p *WebSocketClient) onBinary(bytes []byte) {
	if len(bytes) > 5 {
		b := bytes[1:5]
		clientID := binary.LittleEndian.Uint32(b)
		if cbItem := p.getCallbackByID(clientID); cbItem != nil {
			stream := cbItem.stream
			stream.SetWritePos(0)
			stream.PutBytes(bytes)
			cbItem.ch <- true
		}
	}
}
