package rpc

//
//func TestWebSocketClient_basic(t *testing.T) {
//	assert := newAssert(t)
//	server := NewWebSocketServer()
//	server.AddService("user", newServiceMeta().
//		Echo("sayHello", true, func(
//			ctx *rpcContext,
//			name string,
//		) *rpcReturn {
//			time.Sleep(1000 * time.Millisecond)
//			return ctx.OK("hello " + name)
//		}))
//	server.SetReadTimeoutMS(300)
//	server.StartBackground("0.0.0.0", 12345, "/")
//
//	client := NewWebSocketClient("ws://127.0.0.1:12345/ws")
//
//	finish := make(chan bool, 10000)
//	for i := 0; i < 10000; i++ {
//		go func() {
//			assert(client.SendMessage(
//				"$.user:sayHello",
//				"world",
//			)).Equals("hello world", nil)
//			finish <- true
//		}()
//	}
//
//	for i := 0; i < 10000; i++ {
//		<-finish
//	}
//
//	_ = client.Close()
//	_ = server.Close()
//}
