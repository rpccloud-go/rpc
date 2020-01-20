package rpc

import (
	"fmt"
	"runtime"
	"testing"
)

var reportAssertFail = func(p *Assert) {
	_, file, line, _ := runtime.Caller(2)
	fmt.Printf("%s:%d\n", file, line)
	p.t.Fail()
}

// Assert class
type Assert struct {
	t    interface{ Fail() }
	args []interface{}
}

// NewAssert create new assert class
func NewAssert(t *testing.T) func(args ...interface{}) *Assert {
	return func(args ...interface{}) *Assert {
		return &Assert{
			t:    t,
			args: args,
		}
	}
}

// Fail ...
func (p *Assert) Fail() {
	reportAssertFail(p)
}

// Equals ...
func (p *Assert) Equals(args ...interface{}) {
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
func (p *Assert) Contains(val interface{}) {
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
func (p *Assert) IsNil() {
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
func (p *Assert) IsNotNil() {
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
func (p *Assert) IsTrue() {
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
func (p *Assert) IsFalse() {
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
