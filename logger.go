package rpc

import (
	"fmt"
	"log"
	"sync"
)

const (
	// LogMaskNone this level logs nothing
	LogMaskNone = 0
	// LogMaskFatal this level logs Fatal
	LogMaskFatal = 1
	// LogMaskError this level logs Error
	LogMaskError = 2
	// LogMaskWarn this level logs Warn
	LogMaskWarn = 4
	// LogMaskInfo this level logs Info
	LogMaskInfo = 8
	// LogMaskDebug this level logs Debug
	LogMaskDebug = 16
	// LogMaskAll this level logs Debug, Info, Warn, Error and Fatal
	LogMaskAll = LogMaskFatal |
		LogMaskError |
		LogMaskWarn |
		LogMaskInfo |
		LogMaskDebug
)

// LogSubscription ...
type LogSubscription struct {
	id     int64
	logger *Logger
	Debug  func(msg string)
	Info   func(msg string)
	Warn   func(msg string)
	Error  func(msg string)
	Fatal  func(msg string)
}

// Close ...
func (p *LogSubscription) Close() bool {
	if p.logger == nil {
		return false
	}

	ret := p.logger.removeSubscription(p.id)
	p.id = 0
	p.logger = nil
	p.Debug = nil
	p.Info = nil
	p.Warn = nil
	p.Error = nil
	p.Fatal = nil
	return ret
}

// Logger common logger
type Logger struct {
	level         int
	subscriptions []*LogSubscription
	sync.Mutex
}

// NewLogger create new Logger
func NewLogger() *Logger {
	return &Logger{
		level:         LogMaskAll,
		subscriptions: make([]*LogSubscription, 0, 0),
	}
}

// SetLevel set the log level, such as LogMaskNone LogLevelFatal
// LogLevelError LogLevelWarn LogLevelInfo LogLevelDebug LogMaskAll
func (p *Logger) SetLevel(level int) bool {
	if level >= LogMaskNone && level <= LogMaskAll {
		p.Lock()
		p.level = level
		p.Unlock()
		return true
	}

	return false
}

func (p *Logger) removeSubscription(id int64) bool {
	ret := false
	p.Lock()
	for i := 0; i < len(p.subscriptions); i++ {
		if p.subscriptions[i].id == id {
			array := p.subscriptions
			p.subscriptions = append(array[:i], array[i+1:]...)
			ret = true
			break
		}
	}
	p.Unlock()
	return ret
}

// Subscribe add a subscription to logger
func (p *Logger) Subscribe() *LogSubscription {
	p.Lock()
	subscription := &LogSubscription{
		id:     GetSeed(),
		logger: p,
		Debug:  nil,
		Info:   nil,
		Warn:   nil,
		Error:  nil,
		Fatal:  nil,
	}
	p.subscriptions = append(p.subscriptions, subscription)
	p.Unlock()
	return subscription
}

// Debug log debug
func (p *Logger) Debug(msg string) {
	p.log(LogMaskDebug, " Debug: ", msg)
}

// Debugf log debug
func (p *Logger) Debugf(format string, v ...interface{}) {
	p.log(LogMaskDebug, " Debug: ", fmt.Sprintf(format, v...))
}

// Info log info
func (p *Logger) Info(msg string) {
	p.log(LogMaskInfo, " Info: ", msg)
}

// Infof log info
func (p *Logger) Infof(format string, v ...interface{}) {
	p.log(LogMaskInfo, " Info: ", fmt.Sprintf(format, v...))
}

// Warn log warn
func (p *Logger) Warn(msg string) {
	p.log(LogMaskWarn, " Warn: ", msg)
}

// Warnf log warn
func (p *Logger) Warnf(format string, v ...interface{}) {
	p.log(LogMaskWarn, " Warn: ", fmt.Sprintf(format, v...))
}

// Error log error
func (p *Logger) Error(msg string) {
	p.log(LogMaskError, " Error: ", msg)
}

// Errorf log error
func (p *Logger) Errorf(format string, v ...interface{}) {
	p.log(LogMaskError, " Error: ", fmt.Sprintf(format, v...))
}

// Fatal log fatal
func (p *Logger) Fatal(msg string) {
	p.log(LogMaskFatal, " Fatal: ", msg)
}

// Fatalf log fatal
func (p *Logger) Fatalf(format string, v ...interface{}) {
	p.log(LogMaskFatal, " Fatal: ", fmt.Sprintf(format, v...))
}

func (p *Logger) log(outputLevel int, tag string, msg string) {
	p.Lock()
	level := p.level
	subscriptions := p.subscriptions
	p.Unlock()

	if level&outputLevel > 0 {
		sb := NewStringBuilder()
		sb.AppendString(timeNowISOString())
		sb.AppendString(tag)
		sb.AppendString(msg)
		sb.AppendByte('\n')
		logMsg := sb.String()
		sb.Release()

		if len(subscriptions) == 0 {
			flags := log.Flags()
			log.SetFlags(0)
			log.Printf(logMsg)
			log.SetFlags(flags)
		} else {
			for i := 0; i < len(subscriptions); i++ {
				var fn func(msg string) = nil
				switch outputLevel {
				case LogMaskDebug:
					fn = subscriptions[i].Debug
				case LogMaskInfo:
					fn = subscriptions[i].Info
				case LogMaskWarn:
					fn = subscriptions[i].Warn
				case LogMaskError:
					fn = subscriptions[i].Error
				case LogMaskFatal:
					fn = subscriptions[i].Fatal
				}
				if fn != nil {
					go func() {
						fn(logMsg)
					}()
				} else {
					flags := log.Flags()
					log.SetFlags(0)
					log.Printf(logMsg)
					log.SetFlags(flags)
				}
			}
		}
	}
}
