package rpc

import (
	"fmt"
	"runtime"
	"testing"
)

var reportAssertFail = func(p *rpcAssert) {
	_, file, line, _ := runtime.Caller(2)
	fmt.Printf("%s:%d\n", file, line)
	p.t.Fail()
}

// rpcAssert class
type rpcAssert struct {
	t    interface{ Fail() }
	args []interface{}
}

// newAssert create new assert class
func newAssert(t *testing.T) func(args ...interface{}) *rpcAssert {
	return func(args ...interface{}) *rpcAssert {
		return &rpcAssert{
			t:    t,
			args: args,
		}
	}
}

// Fail ...
func (p *rpcAssert) Fail() {
	reportAssertFail(p)
}

// Equals ...
func (p *rpcAssert) Equals(args ...interface{}) {
	if len(p.args) < 1 {
		reportAssertFail(p)
		return
	}

	if len(p.args) != len(args) {
		reportAssertFail(p)
		return
	}

	for i := 0; i < len(p.args); i++ {
		if !isEquals(p.args[i], args[i]) {
			reportAssertFail(p)
			return
		}
	}
}

// Contains ...
func (p *rpcAssert) Contains(val interface{}) {
	if len(p.args) != 1 {
		reportAssertFail(p)
		return
	}

	if !isContains(p.args[0], val) {
		reportAssertFail(p)
		return
	}
}

// IsNil ...
func (p *rpcAssert) IsNil() {
	if len(p.args) < 1 {
		reportAssertFail(p)
		return
	}

	for i := 0; i < len(p.args); i++ {
		if !isNil(p.args[i]) {
			reportAssertFail(p)
			return
		}
	}
}

// IsNotNil ...
func (p *rpcAssert) IsNotNil() {
	if len(p.args) < 1 {
		reportAssertFail(p)
		return
	}

	for i := 0; i < len(p.args); i++ {
		if isNil(p.args[i]) {
			reportAssertFail(p)
			return
		}
	}
}

// IsTrue ...
func (p *rpcAssert) IsTrue() {
	if len(p.args) < 1 {
		reportAssertFail(p)
		return
	}

	for i := 0; i < len(p.args); i++ {
		if p.args[i] != true {
			reportAssertFail(p)
			return
		}
	}
}

// IsFalse ...
func (p *rpcAssert) IsFalse() {
	if len(p.args) < 1 {
		reportAssertFail(p)
		return
	}

	for i := 0; i < len(p.args); i++ {
		if p.args[i] != false {
			reportAssertFail(p)
			return
		}
	}
}
