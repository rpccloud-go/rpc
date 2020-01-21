package rpc

// NewError create new error
func NewError(message string) Error {
	return &rpcError{
		message: message,
		debug:   "",
	}
}

// NewErrorByDebug create new error
func NewErrorByDebug(message string, debug string) Error {
	return &rpcError{
		message: message,
		debug:   debug,
	}
}

// NewErrorBySystemError add debug segment to the error,
// Note: if err is not Error type, we wrapped it
func NewErrorBySystemError(err error) Error {
	if err == nil {
		return nil
	}

	return &rpcError{
		message: err.Error(),
		debug:   "",
	}
}

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
		sb.AppendFormat("Debug:\n%s\n", addPrefixPerLine(p.debug, "\t"))
	}
	var ret = sb.String()
	sb.Release()
	return ret
}
