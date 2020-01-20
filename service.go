package common

type rpcEchoMeta struct {
	name    string      // the name of echo
	export  bool        // weather echo is export to gateway
	handler interface{} // echo handler
	debug   string      // where the echo add in source file
}

type rpcNodeMeta struct {
	name        string      // the name of child service
	serviceMeta *rpcService // the real service
	debug       string      // where the service add in source file
}

type rpcService struct {
	children []*rpcNodeMeta // all the children node meta pointer
	echos    []*rpcEchoMeta // all the echos meta pointer
	debug    string         // where the service define in source file
	AutoLock
}

// NewService define a new service
func NewService() Service {
	return &rpcService{
		children: make([]*rpcNodeMeta, 0, 0),
		echos:    make([]*rpcEchoMeta, 0, 0),
		debug:    GetStackString(1),
	}
}

// Echo add echo handler
func (p *rpcService) Echo(
	name string,
	export bool,
	handler interface{},
) Service {
	p.DoWithLock(func() {
		// add echo meta
		p.echos = append(p.echos, &rpcEchoMeta{
			name:    name,
			export:  export,
			handler: handler,
			debug:   GetStackString(3),
		})
	})
	return p
}

// AddService add child service
func (p *rpcService) AddService(name string, service Service) Service {
	serviceMeta, ok := service.(*rpcService)
	if !ok {
		serviceMeta = (*rpcService)(nil)
	}

	// lock
	p.DoWithLock(func() {
		// add child meta
		p.children = append(p.children, &rpcNodeMeta{
			name:        name,
			serviceMeta: serviceMeta,
			debug:       GetStackString(3),
		})
	})

	return p
}
