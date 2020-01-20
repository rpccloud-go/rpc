package rpc

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
	"unsafe"
)

var (
	seed                 = int64(10000)
	timeNowPointer       = (unsafe.Pointer)(nil)
	defaultISODateBuffer = []byte{
		0x30, 0x30, 0x30, 0x30, 0x2D, 0x30, 0x30, 0x2D, 0x30, 0x30, 0x54,
		0x30, 0x30, 0x3A, 0x30, 0x30, 0x3A, 0x30, 0x30, 0x2E, 0x30, 0x30, 0x30,
		0x2B, 0x30, 0x30, 0x3A, 0x30, 0x30,
	}
	intToStringCache2 = make([][]byte, 100, 100)
	intToStringCache3 = make([][]byte, 1000, 1000)
	intToStringCache4 = make([][]byte, 10000, 10000)
)

func init() {
	charToASCII := [10]byte{
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x36, 0x37, 0x38, 0x39,
	}
	for i := 0; i < 100; i++ {
		for j := 0; j < 2; j++ {
			intToStringCache2[i] = []byte{
				charToASCII[(i/10)%10],
				charToASCII[i%10],
			}
		}
	}
	for i := 0; i < 1000; i++ {
		for j := 0; j < 3; j++ {
			intToStringCache3[i] = []byte{
				charToASCII[(i/100)%10],
				charToASCII[(i/10)%10],
				charToASCII[i%10],
			}
		}
	}
	for i := 0; i < 10000; i++ {
		for j := 0; j < 4; j++ {
			intToStringCache4[i] = []byte{
				charToASCII[(i/1000)%10],
				charToASCII[(i/100)%10],
				charToASCII[(i/10)%10],
				charToASCII[i%10],
			}
		}
	}

	// New go routine for timer
	go func() {
		tick := time.NewTicker(2 * time.Millisecond)
		for {
			select {
			case t := <-tick.C:
				atomic.StorePointer(&timeNowPointer, unsafe.Pointer(&timeNow{
					timeNS:        t.UnixNano(),
					timeISOString: ConvertToIsoDateString(t),
				}))
			}
		}
	}()
}

type timeNow struct {
	timeNS        int64
	timeISOString string
}

// TimeNowNS get now nanoseconds from 1970-01-01
func TimeNowNS() int64 {
	ret := (*timeNow)(atomic.LoadPointer(&timeNowPointer))
	if ret != nil {
		return ret.timeNS
	}
	return time.Now().UnixNano()
}

// TimeNowMS get now milliseconds from 1970-01-01
func TimeNowMS() int64 {
	return TimeNowNS() / int64(time.Millisecond)
}

// TimeNowISOString get now iso string like this: 2019-09-09T09:47:16.180+08:00
func TimeNowISOString() string {
	ret := (*timeNow)(atomic.LoadPointer(&timeNowPointer))
	if ret != nil {
		return ret.timeISOString
	}
	return ConvertToIsoDateString(time.Now())
}

// TimeSpanFrom get time.Duration from fromNS
func TimeSpanFrom(startNS int64) time.Duration {
	return time.Duration(TimeNowNS() - startNS)
}

// TimeSpanBetween get time.Duration between startNS and endNS
func TimeSpanBetween(startNS int64, endNS int64) time.Duration {
	return time.Duration(endNS - startNS)
}

// GetSeed get int64 seed, it is goroutine safety
func GetSeed() int64 {
	return atomic.AddInt64(&seed, 1)
}

// GetStackString reports the call stack information
func GetStackString(skip uint) string {
	sb := NewStringBuilder()

	idx := uint(1)
	pc, file, line, _ := runtime.Caller(int(skip + idx))

	first := true
	for pc != 0 {
		fn := runtime.FuncForPC(pc)

		if first {
			first = false
			sb.AppendFormat("-%02d %s: %s:%d", idx, fn.Name(), file, line)
		} else {
			sb.AppendFormat("\n-%02d %s: %s:%d", idx, fn.Name(), file, line)
		}

		idx++
		pc, file, line, _ = runtime.Caller(int(skip + idx))
	}

	ret := sb.String()
	sb.Release()
	return ret
}

// GetByteArrayDebugString get the debug string of []byte
func GetByteArrayDebugString(bs []byte) string {
	sb := stringBuilderPool.Get().(*StringBuilder)
	first := true
	for i := 0; i < len(bs); i++ {
		if i%16 == 0 {
			if first {
				first = false
				sb.AppendFormat("%04d: ", i)
			} else {
				sb.AppendFormat("\n%04d: ", i)
			}
		}
		sb.AppendFormat("0x%02X, ", bs[i])
	}
	ret := sb.String()
	sb.Release()
	return ret
}

// ConvertOrdinalToString ...
func ConvertOrdinalToString(n uint) string {
	if n == 0 {
		return ""
	}

	switch n {
	case 1:
		return "1st"
	case 2:
		return "2nd"
	case 3:
		return "3rd"
	default:
		return strconv.Itoa(int(n)) + "th"
	}
}

// ConvertToIsoDateString convert time.Time to iso string
// return format "2019-09-09T09:47:16.180+08:00"
func ConvertToIsoDateString(date time.Time) string {
	buf := make([]byte, 29, 29)
	// copy template
	copy(buf, defaultISODateBuffer)
	// copy year
	year := date.Year()
	if year > 9999 {
		year = 9999
	}
	copy(buf, intToStringCache4[year])
	// copy month
	copy(buf[5:], intToStringCache2[date.Month()])
	// copy date
	copy(buf[8:], intToStringCache2[date.Day()])
	// copy hour
	copy(buf[11:], intToStringCache2[date.Hour()])
	// copy minute
	copy(buf[14:], intToStringCache2[date.Minute()])
	// copy second
	copy(buf[17:], intToStringCache2[date.Second()])
	// copy ms
	copy(buf[20:], intToStringCache3[date.Nanosecond()/1000000])
	// copy timezone
	_, offsetSecond := date.Zone()
	if offsetSecond < 0 {
		buf[23] = '-'
		offsetSecond = -offsetSecond
	}
	copy(buf[24:], intToStringCache2[offsetSecond/3600])
	copy(buf[27:], intToStringCache2[(offsetSecond%3600)/60])
	return string(buf)
}

// IsUTF8Bytes ...
func IsUTF8Bytes(bytes []byte) bool {
	idx := 0
	length := len(bytes)

	for idx < length {
		c := bytes[idx]
		if c < 128 {
			idx++
		} else if c < 224 {
			if (idx+2 > length) ||
				(bytes[idx+1]&0xC0 != 0x80) {
				return false
			}
			idx += 2
		} else if c < 240 {
			if (idx+3 > length) ||
				(bytes[idx+1]&0xC0 != 0x80) ||
				(bytes[idx+2]&0xC0 != 0x80) {
				return false
			}
			idx += 3
		} else if c < 248 {
			if (idx+4 > length) ||
				(bytes[idx+1]&0xC0 != 0x80) ||
				(bytes[idx+2]&0xC0 != 0x80) ||
				(bytes[idx+3]&0xC0 != 0x80) {
				return false
			}
			idx += 4
		} else {
			return false
		}
	}

	return idx == length
}

// GetURLBySchemeHostPortAndPath get the url by connection parameters
func GetURLBySchemeHostPortAndPath(
	scheme string,
	host string,
	port uint16,
	path string,
) string {
	if len(scheme) > 0 && len(host) > 0 {
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		return fmt.Sprintf("%s://%s:%d/%s", scheme, host, port, path)
	}

	return ""
}

// FindLinesByPrefix find the lines start with prefix string
func FindLinesByPrefix(debug string, prefix string) []string {
	ret := make([]string, 0, 0)
	dbgArr := strings.Split(debug, "\n")
	for _, v := range dbgArr {
		if strings.HasPrefix(strings.TrimSpace(v), strings.TrimSpace(prefix)) {
			ret = append(ret, v)
		}
	}
	return ret
}

// AddPrefixPerLine ...
func AddPrefixPerLine(origin string, prefix string) string {
	sb := NewStringBuilder()
	arr := strings.Split(origin, "\n")
	first := true
	for _, v := range arr {
		if first {
			first = false
		} else {
			sb.AppendByte('\n')
		}

		sb.AppendFormat("%s%s", prefix, v)
	}
	ret := sb.String()
	sb.Release()
	return ret
}

func tryToInterfaceArray(v interface{}) ([]interface{}, bool) {
	if isNil(v) {
		return nil, false
	}

	ret := make([]interface{}, 0)
	items := reflect.ValueOf(v)

	switch items.Kind() {
	case reflect.Array:
		fallthrough
	case reflect.Slice:
		for i := 0; i < items.Len(); i++ {
			item := items.Index(i)
			ret = append(ret, item.Interface())
		}
		return ret, true
	default:
		return nil, false
	}
}

func isNil(val interface{}) (ret bool) {
	if val == nil {
		return true
	}

	switch val.(type) {
	case uintptr:
		return val.(uintptr) == 0
	}

	defer func() {
		if e := recover(); e != nil {
			ret = false
		}
	}()

	rv := reflect.ValueOf(val)
	return rv.IsNil()
}

func isEquals(left interface{}, right interface{}) bool {
	if isNil(left) {
		return isNil(right)
	}

	if isNil(right) {
		return false
	}

	if reflect.TypeOf(left) == reflect.TypeOf(right) {
		return reflect.DeepEqual(left, right)
	}

	return false
}

func isContains(left interface{}, right interface{}) bool {
	switch left.(type) {
	case string:
		if rString, ok := right.(string); ok {
			return strings.Contains(left.(string), rString)
		}
		return false
	case []byte:
		if rBytes, ok := right.(byte); ok {
			return bytes.Contains(left.([]byte), []byte{rBytes})
		}
		if rBytes, ok := right.([]byte); ok {
			return bytes.Contains(left.([]byte), rBytes)
		}
		return false
	default:
		if leftArray, ok := tryToInterfaceArray(left); ok {
			if rightArray, ok := tryToInterfaceArray(right); ok {
				if len(rightArray) == 0 {
					return reflect.TypeOf(left) == reflect.TypeOf(right)
				}

				if len(leftArray) == 0 {
					return false
				}

				for i := 0; i+len(rightArray) <= len(leftArray); i++ {
					pos := 0
					for pos < len(rightArray) {
						if !isEquals(leftArray[i+pos], rightArray[pos]) {
							break
						}
						pos++
					}
					if pos == len(rightArray) {
						return true
					}
				}
			} else {
				for i := 0; i < len(leftArray); i++ {
					if isEquals(leftArray[i], right) {
						return true
					}
				}
			}
		}
	}

	return false
}

func getArgumentsErrorPosition(fn reflect.Value) int {
	if fn.Type().NumIn() < 1 {
		return 0
	}

	if fn.Type().In(0) != reflect.ValueOf(nilContext).Type() {
		return 0
	}

	for i := 1; i < fn.Type().NumIn(); i++ {
		argType := fn.Type().In(i)
		switch argType.Kind() {
		case reflect.Uint64:
			continue
		case reflect.Int64:
			continue
		case reflect.Float64:
			continue
		case reflect.Bool:
			continue
		case reflect.String:
			continue
		default:
			if argType == bytesType || argType == arrayType || argType == mapType {
				continue
			}
			return i
		}
	}
	return -1
}

func convertTypeToString(reflectType reflect.Type) string {
	if reflectType == nil {
		return "<nil>"
	}

	switch reflectType {
	case contextType:
		return "rpc.Context"
	case returnType:
		return "rpc.Return"
	case bytesType:
		return "rpc.Bytes"
	case arrayType:
		return "rpc.Array"
	case mapType:
		return "rpc.Map"
	case boolType:
		return "rpc.Bool"
	case int64Type:
		return "rpc.Int"
	case uint64Type:
		return "rpc.Uint"
	case float64Type:
		return "rpc.Float"
	case stringType:
		return "rpc.String"
	default:
		return reflectType.String()
	}
}

func getFuncKind(fn interface{}) (string, bool) {
	if fn == nil {
		return "", false
	}

	reflectFn := reflect.ValueOf(fn)

	// Check echo handler is Func
	if reflectFn.Kind() != reflect.Func {
		return "", false
	}

	if reflectFn.Type().NumIn() < 1 ||
		reflectFn.Type().In(0) != reflect.ValueOf(nilContext).Type() {
		return "", false
	}

	if reflectFn.Type().NumOut() != 1 ||
		reflectFn.Type().Out(0) != reflect.ValueOf(nilReturn).Type() {
		return "", false
	}

	ret := ""
	for i := 1; i < reflectFn.Type().NumIn(); i++ {
		argType := reflectFn.Type().In(i)

		if argType == bytesType {
			ret += "X"
		} else if argType == arrayType {
			ret += "A"
		} else if argType == mapType {
			ret += "M"
		} else {
			switch argType.Kind() {
			case reflect.Int64:
				ret += "I"
			case reflect.Uint64:
				ret += "U"
			case reflect.Bool:
				ret += "B"
			case reflect.Float64:
				ret += "F"
			case reflect.String:
				ret += "S"
			default:
				return "", false
			}
		}
	}
	return ret, true
}

func readStringFromFile(filePath string) (string, error) {
	ret, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(ret), nil
}

func writeStringToFile(s string, filePath string) error {
	if err := os.MkdirAll(path.Dir(filePath), os.ModePerm); err != nil {
		return err
	}

	return ioutil.WriteFile(filePath, []byte(s), 0666)
}
