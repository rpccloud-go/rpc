package rpc

import (
	"os"
	"path"
	"reflect"
	"runtime"
	"sync/atomic"
	"testing"
	"time"
	"unsafe"
)

func TestTimeNowNS(t *testing.T) {
	assert := newAssert(t)

	for i := 0; i < 500000; i++ {
		nowNS := timeNowNS()
		assert(time.Now().UnixNano()-nowNS < int64(20*time.Millisecond)).IsTrue()
		assert(time.Now().UnixNano()-nowNS > int64(-20*time.Millisecond)).IsTrue()
	}

	for i := 0; i < 500; i++ {
		nowNS := timeNowNS()
		assert(time.Now().UnixNano()-nowNS < int64(20*time.Millisecond)).IsTrue()
		assert(time.Now().UnixNano()-nowNS > int64(-20*time.Millisecond)).IsTrue()
		time.Sleep(time.Millisecond)
	}

	// hack timeNowPointer to nil
	atomic.StorePointer(&timeNowPointer, nil)
	for i := 0; i < 500; i++ {
		nowNS := timeNowNS()
		assert(time.Now().UnixNano()-nowNS < int64(20*time.Millisecond)).IsTrue()
		assert(time.Now().UnixNano()-nowNS > int64(-20*time.Millisecond)).IsTrue()
		time.Sleep(time.Millisecond)
	}
}

func TestTimeNowMS(t *testing.T) {
	assert := newAssert(t)
	nowNS := timeNowMS() * int64(time.Millisecond)
	assert(time.Now().UnixNano()-nowNS < int64(20*time.Millisecond)).IsTrue()
	assert(time.Now().UnixNano()-nowNS > int64(-20*time.Millisecond)).IsTrue()
}

func TestTimeNowISOString(t *testing.T) {
	assert := newAssert(t)

	for i := 0; i < 500; i++ {
		if nowNS, err := time.Parse(
			"2006-01-02T15:04:05.999Z07:00",
			timeNowISOString(),
		); err == nil {
			assert(
				time.Now().UnixNano()-nowNS.UnixNano() < int64(20*time.Millisecond),
			).IsTrue()
			assert(
				time.Now().UnixNano()-nowNS.UnixNano() > int64(-20*time.Millisecond),
			).IsTrue()
		} else {
			assert().Fail()
		}
		time.Sleep(time.Millisecond)
	}

	// hack timeNowPointer to nil
	atomic.StorePointer(&timeNowPointer, nil)
	for i := 0; i < 500; i++ {
		if nowNS, err := time.Parse(
			"2006-01-02T15:04:05.999Z07:00",
			timeNowISOString(),
		); err == nil {
			assert(
				time.Now().UnixNano()-nowNS.UnixNano() < int64(20*time.Millisecond),
			).IsTrue()
			assert(
				time.Now().UnixNano()-nowNS.UnixNano() > int64(-20*time.Millisecond),
			).IsTrue()
		} else {
			assert().Fail()
		}
		time.Sleep(time.Millisecond)
	}
}

func TestTimeSpanFrom(t *testing.T) {
	assert := newAssert(t)
	ns := timeNowNS()
	time.Sleep(50 * time.Millisecond)
	dur := timeSpanFrom(ns)
	assert(int64(dur) > int64(40*time.Millisecond)).IsTrue()
	assert(int64(dur) < int64(60*time.Millisecond)).IsTrue()
}

func TestTimeSpanBetween(t *testing.T) {
	assert := newAssert(t)
	start := timeNowNS()
	time.Sleep(50 * time.Millisecond)
	dur := timeSpanBetween(start, timeNowNS())
	assert(int64(dur) > int64(40*time.Millisecond)).IsTrue()
	assert(int64(dur) < int64(60*time.Millisecond)).IsTrue()
}

func TestGetSeed(t *testing.T) {
	assert := newAssert(t)
	seed := getSeed()
	assert(seed > 10000).IsTrue()

	for i := int64(0); i < 1000; i++ {
		assert(getSeed()).Equals(seed + 1 + i)
	}
}

func TestConvertToIsoDateString(t *testing.T) {
	assert := newAssert(t)
	start, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"0001-01-01T00:00:00+00:00",
	)

	for i := 0; i < 1000000; i++ {
		parseTime, err := time.Parse(
			"2006-01-02T15:04:05.999Z07:00",
			convertToIsoDateString(start),
		)
		assert(err).IsNil()
		assert(parseTime.UnixNano()).Equals(start.UnixNano())
		start = start.Add(271099197000000)
	}

	largeTime, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"9998-01-01T00:00:00+00:00",
	)
	largeTime = largeTime.Add(1000000 * time.Hour)
	assert(convertToIsoDateString(largeTime)).
		Equals("9999-01-30T16:00:00.000+00:00")

	time1, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"2222-12-22T11:11:11.333-11:59",
	)
	assert(convertToIsoDateString(time1)).
		Equals("2222-12-22T11:11:11.333-11:59")

	time2, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"2222-12-22T11:11:11.333+11:59",
	)
	assert(convertToIsoDateString(time2)).
		Equals("2222-12-22T11:11:11.333+11:59")

	time3, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"2222-12-22T11:11:11.333+00:00",
	)
	assert(convertToIsoDateString(time3)).
		Equals("2222-12-22T11:11:11.333+00:00")

	time4, _ := time.Parse(
		"2006-01-02T15:04:05.999Z07:00",
		"2222-12-22T11:11:11.333-00:00",
	)
	assert(convertToIsoDateString(time4)).
		Equals("2222-12-22T11:11:11.333+00:00")
}

func TestGetStackString(t *testing.T) {
	assert := newAssert(t)
	assert(findLinesByPrefix(
		getStackString(0),
		"-01",
	)[0]).Contains("TestGetStackString")
	assert(findLinesByPrefix(
		getStackString(0),
		"-01",
	)[0]).Contains("utils_test")
}

func TestFindLinesByPrefix(t *testing.T) {
	assert := newAssert(t)

	ret := findLinesByPrefix("", "")
	assert(len(ret)).Equals(1)
	assert(ret[0]).Equals("")

	ret = findLinesByPrefix("", "hello")
	assert(len(ret)).Equals(0)

	ret = findLinesByPrefix("hello", "dd")
	assert(len(ret)).Equals(0)

	ret = findLinesByPrefix("  hello world", "hello")
	assert(len(ret)).Equals(1)
	assert(ret[0]).Equals("  hello world")

	ret = findLinesByPrefix(" \t hello world", "hello")
	assert(len(ret)).Equals(1)
	assert(ret[0]).Equals(" \t hello world")

	ret = findLinesByPrefix(" \t hello world\nhello\n", "hello")
	assert(len(ret)).Equals(2)
	assert(ret[0]).Equals(" \t hello world")
	assert(ret[1]).Equals("hello")
}

func TestGetByteArrayDebugString(t *testing.T) {
	assert := newAssert(t)
	assert(getByteArrayDebugString([]byte{})).Equals(
		"",
	)
	assert(getByteArrayDebugString([]byte{1, 2})).Equals(
		"0000: 0x01, 0x02, ",
	)
	assert(getByteArrayDebugString(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
	)).Equals(
		"0000: 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, " +
			"0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, ",
	)
	assert(getByteArrayDebugString(
		[]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17},
	)).Equals(
		"0000: 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, " +
			"0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10, \n0016: 0x11, ",
	)
}

func TestGetUrlBySchemeHostPortAndPath(t *testing.T) {
	assert := newAssert(t)

	assert(getURLBySchemeHostPortAndPath("", "127.0.0.1", 8080, "/world")).
		Equals("")
	assert(getURLBySchemeHostPortAndPath("ws", "127.0.0.1", 8080, "")).
		Equals("ws://127.0.0.1:8080/")
	assert(getURLBySchemeHostPortAndPath("ws", "127.0.0.1", 8080, "/")).
		Equals("ws://127.0.0.1:8080/")
	assert(getURLBySchemeHostPortAndPath("ws", "127.0.0.1", 8080, "world")).
		Equals("ws://127.0.0.1:8080/world")
	assert(getURLBySchemeHostPortAndPath("ws", "127.0.0.1", 8080, "/world")).
		Equals("ws://127.0.0.1:8080/world")
}

func TestConvertOrdinalToString(t *testing.T) {
	assert := newAssert(t)

	assert(convertOrdinalToString(0)).Equals("")
	assert(convertOrdinalToString(1)).Equals("1st")
	assert(convertOrdinalToString(2)).Equals("2nd")
	assert(convertOrdinalToString(3)).Equals("3rd")
	assert(convertOrdinalToString(4)).Equals("4th")
	assert(convertOrdinalToString(10)).Equals("10th")
	assert(convertOrdinalToString(100)).Equals("100th")
}

func TestAddPrefixPerLine(t *testing.T) {
	assert := newAssert(t)

	assert(addPrefixPerLine("", "")).Equals("")
	assert(addPrefixPerLine("a", "")).Equals("a")
	assert(addPrefixPerLine("\n", "")).Equals("\n")
	assert(addPrefixPerLine("a\n", "")).Equals("a\n")
	assert(addPrefixPerLine("a\nb", "")).Equals("a\nb")
	assert(addPrefixPerLine("", "-")).Equals("-")
	assert(addPrefixPerLine("a", "-")).Equals("-a")
	assert(addPrefixPerLine("\n", "-")).Equals("-\n-")
	assert(addPrefixPerLine("a\n", "-")).Equals("-a\n-")
	assert(addPrefixPerLine("a\nb", "-")).Equals("-a\n-b")
}

func TestTryToInterfaceArray(t *testing.T) {
	assert := newAssert(t)

	assert(tryToInterfaceArray(nil)).Equals(nil, false)
	assert(tryToInterfaceArray(([]byte)(nil))).Equals(nil, false)
	assert(tryToInterfaceArray([]byte{})).Equals([]interface{}{}, true)
	assert(tryToInterfaceArray([]int{1, 2})).Equals([]interface{}{1, 2}, true)
	assert(tryToInterfaceArray([2]int{})).Equals([]interface{}{0, 0}, true)
	assert(tryToInterfaceArray([2]int{1, 2})).Equals([]interface{}{1, 2}, true)
	assert(tryToInterfaceArray(t)).Equals(nil, false)
}

func TestIsNil(t *testing.T) {
	assert := newAssert(t)
	assert(isNil(nil)).IsTrue()
	assert(isNil(t)).IsFalse()
	assert(isNil(3)).IsFalse()
	assert(isNil(0)).IsFalse()
	assert(isNil(uintptr(0))).IsTrue()
	assert(isNil(uintptr(1))).IsFalse()
	assert(isNil(unsafe.Pointer(nil))).IsTrue()
	assert(isNil(unsafe.Pointer(t))).IsFalse()
}

func TestIsEquals(t *testing.T) {
	assert := newAssert(t)

	assert(isEquals(nil, nil)).IsTrue()
	assert(isEquals(nil, (*rpcAssert)(nil))).IsTrue()
	assert(isEquals(t, nil)).IsFalse()
	assert(isEquals(t, t)).IsTrue()
	assert(isEquals(t, unsafe.Pointer(t))).IsFalse()
}

func TestIsContains(t *testing.T) {
	assert := newAssert(t)

	assert(isContains(nil, nil)).IsFalse()
	assert(isContains(t, nil)).IsFalse()
	assert(isContains(t, t)).IsFalse()

	assert(isContains("hello world", "world")).IsTrue()
	assert(isContains("hello", "world")).IsFalse()
	assert(isContains("hello", t)).IsFalse()

	assert(isContains([]byte{13}, byte(13))).IsTrue()
	assert(isContains([]byte{13}, byte(14))).IsFalse()
	assert(isContains([]byte{13}, 13)).IsFalse()
	assert(isContains([]byte{}, []byte{})).IsTrue()
	assert(isContains([]byte{1, 2, 3}, []byte{})).IsTrue()
	assert(isContains([]byte{1, 2, 3}, []byte{1, 2})).IsTrue()
	assert(isContains([]byte{1, 2, 3}, []byte{2, 3})).IsTrue()
	assert(isContains([]byte{1, 2, 3}, []byte{1, 3})).IsFalse()

	assert(isContains([]uint{13}, uint(13))).IsTrue()
	assert(isContains([]uint{13}, uint(14))).IsFalse()
	assert(isContains([]uint{13}, 13)).IsFalse()
	assert(isContains([]uint{}, []uint{})).IsTrue()
	assert(isContains([]uint{1, 2, 3}, []uint{})).IsTrue()
	assert(isContains([]uint{1, 2, 3}, []uint{1, 2})).IsTrue()
	assert(isContains([]uint{1, 2, 3}, []uint{2, 3})).IsTrue()
	assert(isContains([]uint{1, 2, 3}, []uint{1, 3})).IsFalse()

	assert(isContains([]uint{}, []uint{1})).IsFalse()
	assert(isContains([]uint{}, uint(1))).IsFalse()
}

func TestIsUTF8Bytes(t *testing.T) {
	assert := newAssert(t)

	assert(isUTF8Bytes(([]byte)("abc"))).IsTrue()
	assert(isUTF8Bytes(([]byte)("abc！#@¥#%#%#¥%"))).IsTrue()
	assert(isUTF8Bytes(([]byte)("中文"))).IsTrue()
	assert(isUTF8Bytes(([]byte)("🀄️文👃d"))).IsTrue()
	assert(isUTF8Bytes(([]byte)("🀄️文👃"))).IsTrue()

	assert(isUTF8Bytes(([]byte)(`
    😀 😁 😂 🤣 😃 😄 😅 😆 😉 😊 😋 😎 😍 😘 🥰 😗 😙 😚 ☺️ 🙂 🤗 🤩 🤔 🤨
    🙄 😏 😣 😥 😮 🤐 😯 😪 😫 😴 😌 😛 😜 😝 🤤 😒 😓 😔 😕 🙃 🤑 😲 ☹️ 🙁
    😤 😢 😭 😦 😧 😨 😩 🤯 😬 😰 😱 🥵 🥶 😳 🤪 😵 😡 😠 🤬 😷 🤒 🤕 🤢
    🤡 🥳 🥴 🥺 🤥 🤫 🤭 🧐 🤓 😈 👿 👹 👺 💀 👻 👽 🤖 💩 😺 😸 😹 😻 😼 😽
    👶 👧 🧒 👦 👩 🧑 👨 👵 🧓 👴 👲 👳‍♀️ 👳‍♂️ 🧕 🧔 👱‍♂️ 👱‍♀️ 👨‍🦰 👩‍🦰 👨‍🦱 👩‍🦱 👨‍🦲 👩‍🦲 👨‍🦳
    👩‍🦳 🦸‍♀️ 🦸‍♂️ 🦹‍♀️ 🦹‍♂️ 👮‍♀️ 👮‍♂️ 👷‍♀️ 👷‍♂️ 💂‍♀️ 💂‍♂️ 🕵️‍♀️ 🕵️‍♂️ 👩‍⚕️ 👨‍⚕️ 👩‍🌾 👨‍🌾 👩‍🍳
    👨‍🍳 👩‍🎓 👨‍🎓 👩‍🎤 👨‍🎤 👩‍🏫 👨‍🏫 👩‍🏭 👨‍🏭 👩‍💻 👨‍💻 👩‍💼 👨‍💼 👩‍🔧 👨‍🔧 👩‍🔬 👨‍🔬 👩‍🎨 👨‍🎨 👩‍🚒 👨‍🚒 👩‍✈️ 👨‍✈️ 👩‍🚀
    👩‍⚖️ 👨‍⚖️ 👰 🤵 👸 🤴 🤶 🎅 🧙‍♀️ 🧙‍♂️ 🧝‍♀️ 🧝‍♂️ 🧛‍♀️ 🧛‍♂️ 🧟‍♀️ 🧟‍♂️ 🧞‍♀️ 🧞‍♂️ 🧜‍♀️
    🧜‍♂️ 🧚‍♀️ 🧚‍♂️ 👼 🤰 🤱 🙇‍♀️ 🙇‍♂️ 💁‍♀️ 💁‍♂️ 🙅‍♀️ 🙅‍♂️ 🙆‍♀️ 🙆‍♂️ 🙋‍♀️ 🙋‍♂️ 🤦‍♀️ 🤦‍♂️
    🤷‍♀️ 🤷‍♂️ 🙎‍♀️ 🙎‍♂️ 🙍‍♀️ 🙍‍♂️ 💇‍♀️ 💇‍♂️ 💆‍♀️ 💆‍♂️ 🧖‍♀️ 🧖‍♂️ 💅 🤳 💃 🕺 👯‍♀️ 👯‍♂️
    🕴 🚶‍♀️ 🚶‍♂️ 🏃‍♀️ 🏃‍♂️ 👫 👭 👬 💑 👩‍❤️‍👩 👨‍❤️‍👨 💏 👩‍❤️‍💋‍👩 👨‍❤️‍💋‍👨 👪 👨‍👩‍👧 👨‍👩‍👧‍👦 👨‍👩‍👦‍👦
    👨‍👩‍👧‍👧 👩‍👩‍👦 👩‍👩‍👧 👩‍👩‍👧‍👦 👩‍👩‍👦‍👦 👩‍👩‍👧‍👧 👨‍👨‍👦 👨‍👨‍👧 👨‍👨‍👧‍👦 👨‍👨‍👦‍👦 👨‍👨‍👧‍👧 👩‍👦 👩‍👧 👩‍👧‍👦 👩‍👦‍👦 👩‍👧‍👧 👨‍👦 👨‍👧 👨‍👧‍👦 👨‍👦‍👦 👨‍👧‍👧 🤲 👐
    👎 👊 ✊ 🤛 🤜 🤞 ✌️ 🤟 🤘 👌 👈 👉 👆 👇 ☝️ ✋ 🤚 🖐 🖖 👋 🤙 💪 🦵 🦶
    💍 💄 💋 👄 👅 👂 👃 👣 👁 👀 🧠 🦴 🦷 🗣 👤 👥
  `))).IsTrue()

	assert(isUTF8Bytes([]byte{0xC1})).IsFalse()
	assert(isUTF8Bytes([]byte{0xC1, 0x01})).IsFalse()

	assert(isUTF8Bytes([]byte{0xE1, 0x80})).IsFalse()
	assert(isUTF8Bytes([]byte{0xE1, 0x01, 0x81})).IsFalse()
	assert(isUTF8Bytes([]byte{0xE1, 0x80, 0x01})).IsFalse()

	assert(isUTF8Bytes([]byte{0xF1, 0x80, 0x80})).IsFalse()
	assert(isUTF8Bytes([]byte{0xF1, 0x70, 0x80, 0x80})).IsFalse()
	assert(isUTF8Bytes([]byte{0xF1, 0x80, 0x70, 0x80})).IsFalse()
	assert(isUTF8Bytes([]byte{0xF1, 0x80, 0x80, 0x70})).IsFalse()

	assert(isUTF8Bytes([]byte{0xFF, 0x80, 0x80, 0x70})).IsFalse()
}

func TestGetArgumentsErrorPosition(t *testing.T) {
	assert := newAssert(t)

	fn1 := func() {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn1))).Equals(0)
	fn2 := func(_ chan bool) {}

	assert(getArgumentsErrorPosition(reflect.ValueOf(fn2))).Equals(0)
	fn3 := func(ctx Context, _ bool, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn3))).Equals(2)
	fn4 := func(ctx Context, _ int64, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn4))).Equals(2)
	fn5 := func(ctx Context, _ uint64, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn5))).Equals(2)
	fn6 := func(ctx Context, _ float64, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn6))).Equals(2)
	fn7 := func(ctx Context, _ string, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn7))).Equals(2)
	fn8 := func(ctx Context, _ Bytes, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn8))).Equals(2)
	fn9 := func(ctx Context, _ Array, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn9))).Equals(2)
	fn10 := func(ctx Context, _ Map, _ chan bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn10))).Equals(2)

	fn11 := func(ctx Context, _ bool) {}
	assert(getArgumentsErrorPosition(reflect.ValueOf(fn11))).Equals(-1)
}

func TestConvertTypeToString(t *testing.T) {
	assert := newAssert(t)
	assert(convertTypeToString(nil)).Equals("<nil>")
	assert(convertTypeToString(bytesType)).Equals("rpc.Bytes")
	assert(convertTypeToString(arrayType)).Equals("rpc.Array")
	assert(convertTypeToString(mapType)).Equals("rpc.Map")
	assert(convertTypeToString(boolType)).Equals("rpc.Bool")
	assert(convertTypeToString(int64Type)).Equals("rpc.Int64")
	assert(convertTypeToString(uint64Type)).Equals("rpc.Uint64")
	assert(convertTypeToString(float64Type)).Equals("rpc.Float64")
	assert(convertTypeToString(stringType)).Equals("rpc.String")
	assert(convertTypeToString(contextType)).Equals("rpc.Context")
	assert(convertTypeToString(returnType)).Equals("rpc.Return")
	assert(convertTypeToString(reflect.ValueOf(make(chan bool)).Type())).
		Equals("chan bool")
}

func TestGetFuncKind(t *testing.T) {
	assert := newAssert(t)

	assert(getFuncKind(nil)).Equals("", false)
	assert(getFuncKind(3)).Equals("", false)
	fn1 := func() {}
	assert(getFuncKind(fn1)).Equals("", false)
	fn2 := func(_ chan bool) {}
	assert(getFuncKind(fn2)).Equals("", false)
	fn3 := func(ctx Context, _ bool) Return { return nilReturn }
	assert(getFuncKind(fn3)).Equals("B", true)
	fn4 := func(ctx Context, _ int64) Return { return nilReturn }
	assert(getFuncKind(fn4)).Equals("I", true)
	fn5 := func(ctx Context, _ uint64) Return { return nilReturn }
	assert(getFuncKind(fn5)).Equals("U", true)
	fn6 := func(ctx Context, _ float64) Return { return nilReturn }
	assert(getFuncKind(fn6)).Equals("F", true)
	fn7 := func(ctx Context, _ string) Return { return nilReturn }
	assert(getFuncKind(fn7)).Equals("S", true)
	fn8 := func(ctx Context, _ Bytes) Return { return nilReturn }
	assert(getFuncKind(fn8)).Equals("X", true)
	fn9 := func(ctx Context, _ Array) Return { return nilReturn }
	assert(getFuncKind(fn9)).Equals("A", true)
	fn10 := func(ctx Context, _ Map) Return { return nilReturn }
	assert(getFuncKind(fn10)).Equals("M", true)

	fn11 := func(ctx Context) Return { return nilReturn }
	assert(getFuncKind(fn11)).Equals("", true)

	// no return
	fn12 := func(ctx Context, _ bool) {}
	assert(getFuncKind(fn12)).Equals("", false)

	// value type not supported
	fn13 := func(ctx Context, _ chan bool) Return { return nilReturn }
	assert(getFuncKind(fn13)).Equals("", false)

	fn14 := func(
		ctx Context,
		_ bool, _ int64, _ uint64, _ float64, _ string,
		_ Bytes, _ Array, _ Map,
	) Return {
		return nilReturn
	}
	assert(getFuncKind(fn14)).Equals("BIUFSXAM", true)
}

func TestReadStringFromFile(t *testing.T) {
	assert := newAssert(t)
	_, file, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(file), "_tmp_")

	assert(writeStringToFile("hello", path.Join(dir, "test"))).IsNil()
	assert(readStringFromFile(path.Join(dir, "test"))).Equals("hello", nil)
	ret, err := readStringFromFile(path.Join(dir, "notExist"))
	assert(ret).Equals("")
	assert(err).IsNotNil()

	_ = os.RemoveAll(dir)
}

func TestWriteStringToFile(t *testing.T) {
	assert := newAssert(t)
	_, file, _, _ := runtime.Caller(0)
	dir := path.Join(path.Dir(file), "_tmp_")

	assert(writeStringToFile("hello", path.Join(dir, "test"))).IsNil()
	assert(writeStringToFile("hello", path.Join(dir, "test", "abc"))).IsNotNil()

	_ = os.RemoveAll(dir)
}
