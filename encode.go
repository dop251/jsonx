package jsonx

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"reflect"
	"time"
	"encoding/base64"
)

const (
	MAX_SAFE_INTEGER = 1<<53 - 1
	MIN_SAFE_INTEGER = -(1<<53 - 1)
)

type writer interface {
	io.Writer
	io.ByteWriter
	WriteString(string) (int, error)
	WriteRune(rune) (int, error)
	Flush() error
}

type noopFlusher struct {
	writer
}

type memWriter struct {
	bytes.Buffer
}

type Encoder struct {
	w              writer
	base64Encoder  io.WriteCloser
	pretty         bool
	prefix, indent string

	level int
}

func (noopFlusher) Flush() error {
	return nil
}

func (*memWriter) Flush() error {
	return nil
}

func newWriter(w io.Writer) writer {
	if w1, ok := w.(writer); ok {
		return noopFlusher{w1}
	} else {
		return bufio.NewWriter(w)
	}
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: newWriter(w),
	}
}

func NewEncoderIndent(w io.Writer, prefix, indent string) *Encoder {
	return &Encoder{
		w:      newWriter(w),
		pretty: true,
		prefix: prefix,
		indent: indent,
	}
}

func Marshal(v interface{}) ([]byte, error) {
	var w memWriter
	e := Encoder{w: &w}
	err := e.Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	var w memWriter
	e := Encoder{w: &w, pretty: true, prefix: prefix, indent: indent}
	err := e.Encode(v)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (e *Encoder) Encode(v interface{}) error {
	err := e.encodeValue(v)
	if err != nil {
		return err
	}

	return e.w.Flush()
}

func (e *Encoder) encodeValue(v interface{}) (err error) {
	switch v := v.(type) {
	case string:
		err = e.encodeString(v)
	case map[string]interface{}:
		err = e.encodeMap(v)
	case []interface{}:
		err = e.encodeArray(v)
	case []byte:
		err = e.encodeBytes(v)
	case int:
		err = e.encodeInt(v)
	case nil:
		_, err = e.w.WriteString("null")
	case bool:
		if v {
			_, err = e.w.WriteString("true")
		} else {
			_, err = e.w.WriteString("false")
		}
	case time.Time:
		err = e.encodeTime(v)
	case net.IP:
		err = e.encodeIP(v)
	case net.TCPAddr:
		err = e.encodeIPPort(v.IP, v.Port)
	case *net.TCPAddr:
		err = e.encodeIPPort(v.IP, v.Port)
	case net.UDPAddr:
		err = e.encodeIPPort(v.IP, v.Port)
	case *net.UDPAddr:
		err = e.encodeIPPort(v.IP, v.Port)
	case uint:
		err = e.encodeUInt(v)
	case int32:
		err = e.encodeInt32(v)
	case uint32:
		err = e.encodeUInt32(v)
	case int64:
		err = e.encodeInt64(v)
	case uint64:
		err = e.encodeUInt64(v)
	case int8:
		err = e.encodeInt8(v)
	case uint8:
		err = e.encodeUInt8(v)
	case int16:
		err = e.encodeInt16(v)
	case uint16:
		err = e.encodeUInt16(v)
	case float64:
		err = e.encodeFloat64(v)
	default:
		switch v1 := reflect.ValueOf(v); v1.Kind() {
		case reflect.Slice:
			err = e.encodeSlice(v1)
		default:
			err = fmt.Errorf("Unsupported value type: %T", v)
		}
	}

	return
}

func (e *Encoder) encodeTime(t time.Time) error {
	_, err := fmt.Fprintf(e.w, "datetime(\"%s\")", t.Format(time.RFC3339))
	return err
}

func (e *Encoder) encodeIP(ip net.IP) error {
	_, err := fmt.Fprintf(e.w, "ip(\"%s\")", ip.String())
	return err
}

func (e *Encoder) encodeIPPort(ip net.IP, port int) (err error) {
	if ip4 := ip.To4(); ip4 != nil {
		_, err = fmt.Fprintf(e.w, "ipport(\"%s:%d\")", ip4.String(), port)
	} else {
		_, err = fmt.Fprintf(e.w, "ipport(\"[%s]:%d\")", ip.String(), port)
	}

	return
}

func (e *Encoder) encodeFloat64(v float64) error {
	_, err := e.w.WriteString(strconv.FormatFloat(v, 'g', -1, 64))
	return err
}

func (e *Encoder) encodeInt(v int) error {
	_, err := e.w.WriteString("int(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.Itoa(v))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeUInt(v uint) error {
	_, err := e.w.WriteString("uint(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatUint(uint64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeInt8(v int8) error {
	_, err := e.w.WriteString("int8(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatInt(int64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeInt16(v int16) error {
	_, err := e.w.WriteString("int16(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatInt(int64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeInt32(v int32) error {
	_, err := e.w.WriteString("int32(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatInt(int64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeInt64(v int64) error {
	_, err := e.w.WriteString("int64(")
	if err != nil {
		return err
	}
	if v > MAX_SAFE_INTEGER || v < MIN_SAFE_INTEGER {
		err := e.w.WriteByte('"')
		if err != nil {
			return err
		}
	}
	_, err = e.w.WriteString(strconv.FormatInt(int64(v), 10))
	if err != nil {
		return err
	}
	if v > MAX_SAFE_INTEGER || v < MIN_SAFE_INTEGER {
		err := e.w.WriteByte('"')
		if err != nil {
			return err
		}
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeUInt8(v uint8) error {
	_, err := e.w.WriteString("uint8(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatUint(uint64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeUInt16(v uint16) error {
	_, err := e.w.WriteString("uint16(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatUint(uint64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeUInt32(v uint32) error {
	_, err := e.w.WriteString("uint32(")
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(strconv.FormatUint(uint64(v), 10))
	if err != nil {
		return err
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) encodeUInt64(v uint64) error {
	_, err := e.w.WriteString("uint64(")
	if err != nil {
		return err
	}
	if v > MAX_SAFE_INTEGER {
		err := e.w.WriteByte('"')
		if err != nil {
			return err
		}
	}
	_, err = e.w.WriteString(strconv.FormatUint(uint64(v), 10))
	if err != nil {
		return err
	}
	if v > MAX_SAFE_INTEGER {
		err := e.w.WriteByte('"')
		if err != nil {
			return err
		}
	}
	return e.w.WriteByte(')')
}

func (e *Encoder) writeIndent() error {
	err := e.w.WriteByte('\n')
	if err != nil {
		return err
	}
	_, err = e.w.WriteString(e.prefix)
	if err != nil {
		return err
	}

	for i := 0; i < e.level; i++ {
		_, err = e.w.WriteString(e.indent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (e *Encoder) encodeMap(m map[string]interface{}) error {
	keys := make([]string, len(m))
	i := 0
	for key := range m {
		keys[i] = key
		i++
	}
	sort.Strings(keys)
	e.w.WriteByte('{')
	if e.pretty {
		e.level++
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}
	first := true
	for _, k := range keys {
		if !first {
			err := e.w.WriteByte(',')
			if err != nil {
				return err
			}
			if e.pretty {
				err = e.writeIndent()
				if err != nil {
					return err
				}
			}
		} else {
			first = false
		}
		v := m[k]
		err := e.encodeKey(k)
		if err != nil {
			return err
		}
		err = e.w.WriteByte(':')
		if err != nil {
			return err
		}
		if e.pretty {
			err = e.w.WriteByte(' ')
			if err != nil {
				return err
			}
		}
		err = e.encodeValue(v)
		if err != nil {
			return err
		}
	}

	if e.pretty {
		e.level--
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}

	return e.w.WriteByte('}')
}

func (e *Encoder) encodeKey(key string) error {
	if len(key) > 0 {
		if c := key[0]; c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			for i := 1; i < len(key); i++ {
				if c := key[i]; c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' {

				} else {
					goto notatom
				}
			}

			_, err := e.w.WriteString(key)
			return err
		}
	}

notatom:
	return e.encodeString(key)
}

func (e *Encoder) encodeArray(a []interface{}) error {
	err := e.w.WriteByte('[')
	if err != nil {
		return err
	}
	if e.pretty {
		e.level++
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}
	first := true
	for _, v := range a {
		if !first {
			err = e.w.WriteByte(',')
			if err != nil {
				return err
			}
			if e.pretty {
				err = e.writeIndent()
				if err != nil {
					return err
				}
			}
		} else {
			first = false
		}
		err = e.encodeValue(v)
		if err != nil {
			return err
		}
	}

	if e.pretty {
		e.level--
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}

	return e.w.WriteByte(']')
}

func (e *Encoder) encodeSlice(s reflect.Value) error {
	err := e.w.WriteByte('[')
	if err != nil {
		return err
	}
	if e.pretty {
		e.level++
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}
	first := true
	for i := 0; i < s.Len(); i++ {
		if !first {
			err = e.w.WriteByte(',')
			if err != nil {
				return err
			}
			if e.pretty {
				err = e.writeIndent()
				if err != nil {
					return err
				}
			}
		} else {
			first = false
		}
		err = e.encodeValue(s.Index(i).Interface())
		if err != nil {
			return err
		}
	}

	if e.pretty {
		e.level--
		err := e.writeIndent()
		if err != nil {
			return err
		}
	}

	return e.w.WriteByte(']')
}

func (e *Encoder) encodeBytes(b []byte) error {
	_, err := e.w.WriteString("bytes(\"")
	if err != nil {
		return err
	}
	if e.base64Encoder == nil {
		e.base64Encoder = base64.NewEncoder(base64.StdEncoding, e.w)
	}
	_, err = e.base64Encoder.Write(b)
	if err != nil {
		return err
	}
	err = e.base64Encoder.Close()
	if err != nil {
		return err
	}
	_, err = e.w.WriteString("\")")
	return err
}

func (e *Encoder) encodeString(str string) (err error) {
	err = e.w.WriteByte('"')
	for _, c := range str {
		switch c {
		case '\\', '"', '\r', '\n', '\f', '\t':
			err = e.w.WriteByte('\\')
			if err != nil {
				return
			}
		}
		switch c {
		case '\r':
			c = 'r'
		case '\n':
			c = 'n'
		case '\f':
			c = 'f'
		case '\t':
			c = 't'
		}
		_, err = e.w.WriteRune(c)
		if err != nil {
			return err
		}
	}
	return e.w.WriteByte('"')
}
