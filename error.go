package rpc

// RPCError ...
type RPCError interface {
	GetMessage() string
	GetDebug() string
	AddDebug(debug string)
	Error() string
}

// NewRPCError create new error
func NewRPCError(message string) RPCError {
	return &rpcError{
		message: message,
		debug:   "",
	}
}

// NewRPCErrorByDebug create new error
func NewRPCErrorByDebug(message string, debug string) RPCError {
	return &rpcError{
		message: message,
		debug:   debug,
	}
}

// NewRPCErrorByError add debug segment to the error,
// Note: if err is not Error type, we wrapped it
func NewRPCErrorByError(err error) RPCError {
	if err == nil {
		return nil
	}

	return &rpcError{
		message: err.Error(),
		debug:   "",
	}
}

// NewRPCError ...
type rpcError struct {
	message string
	debug   string
}

func (p *rpcError) GetMessage() string {
	return p.message
}

func (p *rpcError) GetDebug() string {
	return p.debug
}

func (p *rpcError) AddDebug(debug string) {
	if p.debug != "" {
		p.debug += "\n"
	}
	p.debug += debug
}

func (p *rpcError) Error() string {
	sb := NewStringBuilder()
	if len(p.message) > 0 {
		sb.AppendFormat("%s\n", p.message)
	}

	if len(p.debug) > 0 {
		sb.AppendFormat("Debug:\n%s\n", AddPrefixPerLine(p.debug, "\t"))
	}
	var ret = sb.String()
	sb.Release()
	return ret
}
