package rpc

import (
	"bytes"
	"log"
	"os"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLogSubscription_Close(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	subscription := logger.Subscribe()
	subscription.Debug = func(msg string) {}
	subscription.Info = func(msg string) {}
	subscription.Warn = func(msg string) {}
	subscription.Error = func(msg string) {}
	subscription.Fatal = func(msg string) {}
	assert(subscription.Close()).IsTrue()
	assert(subscription.id == 0).IsTrue()
	assert(subscription.logger).IsNil()
	assert(subscription.Debug).IsNil()
	assert(subscription.Info).IsNil()
	assert(subscription.Warn).IsNil()
	assert(subscription.Error).IsNil()
	assert(subscription.Fatal).IsNil()
	assert(len(logger.subscriptions)).Equals(0)

	subscription1 := logger.Subscribe()
	subscription2 := logger.Subscribe()
	assert(subscription1.Close()).IsTrue()
	assert(subscription1.Close()).IsFalse()
	assert(subscription2.Close()).IsTrue()

	subscription = logger.Subscribe()
	subscription.logger = nil
	assert(subscription.Close()).IsFalse()
}

func TestNewLogger(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()
	assert(logger.level).Equals(LogMaskAll)
	assert(len(logger.subscriptions)).Equals(0)
}

func TestLogger_SetLevel(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	assert(logger.SetLevel(-1)).IsFalse()
	assert(logger.level).Equals(LogMaskAll)

	assert(logger.SetLevel(32)).IsFalse()
	assert(logger.level).Equals(LogMaskAll)

	assert(logger.SetLevel(0)).IsTrue()
	assert(logger.level).Equals(0)

	assert(logger.SetLevel(31)).IsTrue()
	assert(logger.level).Equals(31)
	// test all level and logs
	fnTestLogLevel := func(level int) int {
		logger := NewLogger()
		logger.SetLevel(level)
		ret := int32(0)
		subscription := logger.Subscribe()
		subscription.Debug = func(msg string) {
			if strings.Contains(msg, "message") &&
				strings.Contains(msg, "Debug") {
				atomic.AddInt32(&ret, LogMaskDebug)
			}
		}
		subscription.Info = func(msg string) {
			if strings.Contains(msg, "message") &&
				strings.Contains(msg, "Info") {
				atomic.AddInt32(&ret, LogMaskInfo)
			}
		}
		subscription.Warn = func(msg string) {
			if strings.Contains(msg, "message") &&
				strings.Contains(msg, "Warn") {
				atomic.AddInt32(&ret, LogMaskWarn)
			}
		}
		subscription.Error = func(msg string) {
			if strings.Contains(msg, "message") &&
				strings.Contains(msg, "Error") {
				atomic.AddInt32(&ret, LogMaskError)
			}
		}
		subscription.Fatal = func(msg string) {
			if strings.Contains(msg, "message") &&
				strings.Contains(msg, "Fatal") {
				atomic.AddInt32(&ret, LogMaskFatal)
			}
		}

		logger.Debug("message")
		logger.Info("message")
		logger.Warn("message")
		logger.Error("message")
		logger.Fatal("message")
		time.Sleep(30 * time.Millisecond)
		subscription.Close()

		return int(atomic.LoadInt32(&ret))
	}

	assert(fnTestLogLevel(-1)).Equals(31)
	for i := 0; i < 32; i++ {
		assert(fnTestLogLevel(i)).Equals(i)
	}
	assert(fnTestLogLevel(32)).Equals(31)
}

func TestLogger_Subscribe(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	subscription := logger.Subscribe()
	assert(subscription.id > 0).IsTrue()
	assert(subscription.logger).Equals(logger)
	assert(len(logger.subscriptions)).Equals(1)

	assert(subscription.Close()).IsTrue()
	assert(subscription.id == 0).IsTrue()
	assert(subscription.logger).IsNil()
	assert(len(logger.subscriptions)).Equals(0)

	assert(subscription.Close()).IsFalse()
}

func TestLogger_log(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf bytes.Buffer
	log.SetOutput(&buf)
	logger.Subscribe()

	logger.Debug("")
	assert(buf.String()).Contains("Debug")
	logger.Info("")
	assert(buf.String()).Contains("Info")
	logger.Warn("")
	assert(buf.String()).Contains("Warn")
	logger.Error("")
	assert(buf.String()).Contains("Error")
	logger.Fatal("")
	assert(buf.String()).Contains("Fatal")

	log.SetOutput(os.Stderr)
}

func TestLogger_Debug(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf1 bytes.Buffer
	log.SetOutput(&buf1)
	logger.Debug("message")
	regex1 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Debug:(\\s)message\n$"
	assert(regexp.MatchString(regex1, buf1.String())).Equals(true, nil)

	var buf3 bytes.Buffer
	log.SetOutput(&buf3)

	logger.Debugf("message")
	regex3 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Debug:(\\s)message\n$"
	assert(regexp.MatchString(regex3, buf3.String())).Equals(true, nil)

	var buf4 bytes.Buffer
	log.SetOutput(&buf4)
	logger.Debugf("message %d", 1)
	regex4 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Debug:(\\s)message 1\n$"
	assert(regexp.MatchString(regex4, buf4.String())).Equals(true, nil)

	log.SetOutput(os.Stderr)
}

func TestLogger_Info(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf1 bytes.Buffer
	log.SetOutput(&buf1)
	logger.Info("message")
	regex1 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Info:(\\s)message\n$"
	assert(regexp.MatchString(regex1, buf1.String())).Equals(true, nil)

	var buf3 bytes.Buffer
	log.SetOutput(&buf3)
	logger.Infof("message")
	regex3 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Info:(\\s)message\n$"
	assert(regexp.MatchString(regex3, buf3.String())).Equals(true, nil)

	var buf4 bytes.Buffer
	log.SetOutput(&buf4)
	logger.Infof("message %d", 1)
	regex4 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Info:(\\s)message 1\n$"
	assert(regexp.MatchString(regex4, buf4.String())).Equals(true, nil)

	log.SetOutput(os.Stderr)
}

func TestLogger_Warn(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf1 bytes.Buffer
	log.SetOutput(&buf1)
	logger.Warn("message")
	regex1 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Warn:(\\s)message\n$"
	assert(regexp.MatchString(regex1, buf1.String())).Equals(true, nil)

	var buf3 bytes.Buffer
	log.SetOutput(&buf3)
	logger.Warnf("message")
	regex3 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Warn:(\\s)message\n$"
	assert(regexp.MatchString(regex3, buf3.String())).Equals(true, nil)

	var buf4 bytes.Buffer
	log.SetOutput(&buf4)
	logger.Warnf("message %d", 1)
	regex4 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Warn:(\\s)message 1\n$"
	assert(regexp.MatchString(regex4, buf4.String())).Equals(true, nil)

	log.SetOutput(os.Stderr)
}

func TestLogger_Error(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf1 bytes.Buffer
	log.SetOutput(&buf1)
	logger.Error("message")
	regex1 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Error:(\\s)message\n$"
	assert(regexp.MatchString(regex1, buf1.String())).Equals(true, nil)

	var buf3 bytes.Buffer
	log.SetOutput(&buf3)
	logger.Errorf("message")
	regex3 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Error:(\\s)message\n$"
	assert(regexp.MatchString(regex3, buf3.String())).Equals(true, nil)

	var buf4 bytes.Buffer
	log.SetOutput(&buf4)
	logger.Errorf("message %d", 1)
	regex4 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Error:(\\s)message 1\n$"
	assert(regexp.MatchString(regex4, buf4.String())).Equals(true, nil)

	log.SetOutput(os.Stderr)
}

func TestLogger_Fatal(t *testing.T) {
	assert := newAssert(t)
	logger := NewLogger()

	var buf1 bytes.Buffer
	log.SetOutput(&buf1)
	logger.Fatal("message")
	regex1 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Fatal:(\\s)message\n$"
	assert(regexp.MatchString(regex1, buf1.String())).Equals(true, nil)

	var buf3 bytes.Buffer
	log.SetOutput(&buf3)
	logger.Fatalf("message")
	regex3 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Fatal:(\\s)message\n$"
	assert(regexp.MatchString(regex3, buf3.String())).Equals(true, nil)

	var buf4 bytes.Buffer
	log.SetOutput(&buf4)
	logger.Fatalf("message %d", 1)
	regex4 := "^\\d{4}-\\d{2}-\\d{2}T\\d{2}:\\d{2}:\\d{2}.\\d{3}" +
		"\\+\\d{2}:\\d{2}(\\s)Fatal:(\\s)message 1\n$"
	assert(regexp.MatchString(regex4, buf4.String())).Equals(true, nil)

	log.SetOutput(os.Stderr)
}
