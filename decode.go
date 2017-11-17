package jsonx

import (
	"net"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// Decoder is the object that holds the state of the decoding
type Decoder struct {
	pos       int
	end       int
	data      []byte
	sdata     string
	usestring bool
}

// NewDecoder creates new Decoder from the JSON-encoded data
func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		data: data,
		end:  len(data),
	}
}

// AllocString pre-allocates a string version of the data before starting
// to decode the data.
// It is used to make the decode operation faster(see below) by doing one
// allocation operation for string conversion(from bytes), and then uses
// "slicing" to create non-escaped strings in the "Decoder.string" method.
// However, string is a read-only slice, and since the slice references the
// original array, as long as the slice is kept around, the garbage collector
// can't release the array.
// For this reason, you want to use this method only when the Decoder's result
// is a "read-only" or you are adding more elements to it. see example below.
//
// Here are the improvements:
//
//	small payload  - 0.13~ time faster, does 0.45~ less memory allocations but
// 			 the total number of bytes that are allocated is 0.03~ bigger
//
// 	medium payload - 0.16~ time faster, does 0.5~ less memory allocations but
// 			 the total number of bytes that are allocated is 0.05~ bigger
//
// 	large payload  - 0.13~ time faster, does 0.50~ less memory allocations but
// 			 the total number of bytes that are allocated is 0.02~ bigger
//
// Here is an example to illustrate when you don't want to use this method
//
// 	str := fmt.Sprintf(`{"foo": "bar", "baz": "%s"}`, strings.Repeat("#", 1024 * 1024))
//	dec := djson.NewDecoder([]byte(str))
// 	dec.AllocString()
// 	ev, err := dec.DecodeObject()
//
// 	// inspect memory stats here; MemStats.Alloc ~= 1M
//
// 	delete(ev, "baz") // or ev["baz"] = "qux"
//
// 	// inspect memory stats again; MemStats.Alloc ~= 1M
// 	// it means that the chunk that was located in the "baz" value is not freed
//
func (d *Decoder) AllocString() {
	d.sdata = string(d.data)
	d.usestring = true
}

// Decode parses the JSONX-encoded data and returns an interface value.
// The interface value could be one of these:
//
//	bool, for booleans
//	float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, for numbers
//	string, for strings
//  net.IP for IP addresses (ip("1.2.3.4") or ip("fd00::1"))
//  net.TCPAddr for ip/port pairs (ipport("1.2.3.4:5678") or ipport("[fd00::1]:5678")
//  time.Time for timestamps (datetime("2006-01-02T15:04:05Z07:00"))
//	[]interface{}, for arrays
//	map[string]interface{}, for objects
//	nil for null
//
// If any extra non-space characters found after decoding the top level value, the decoded value and the error
// are returned allowing to implement non-greedy decoding.
func (d *Decoder) Decode() (interface{}, error) {
	d.skipSpaces()
	val, err := d.any()
	if err != nil {
		return nil, err
	}
	if d.skipSpaces(); d.pos < d.end {
		return val, &ExtraDataError{d.pos}
	}
	return val, nil
}

// DecodeObject is the same as Decode but it returns map[string]interface{}.
func (d *Decoder) DecodeObject() (map[string]interface{}, error) {
	if c := d.skipSpaces(); c != '{' {
		return nil, d.error(c, "looking for beginning of object")
	}
	val, err := d.object()
	if err != nil {
		return nil, err
	}
	if d.skipSpaces(); d.pos < d.end {
		return val, &ExtraDataError{d.pos}
	}
	return val, nil
}

// DecodeArray is the same as Decode but it returns []interface{}.
func (d *Decoder) DecodeArray() ([]interface{}, error) {
	if c := d.skipSpaces(); c != '[' {
		return nil, d.error(c, "looking for beginning of array")
	}
	val, err := d.array()
	if err != nil {
		return nil, err
	}
	if d.skipSpaces(); d.pos < d.end {
		return val, &ExtraDataError{d.pos}
	}
	return val, nil
}

// any used to decode any valid JSONX value, and returns an
// interface{} that holds the actual data
func (d *Decoder) any() (interface{}, error) {
	if d.pos >= d.end {
		return nil, d.error(0, "looking for beginning of value")
	}

	switch c := d.data[d.pos]; c {
	case '"':
		return d.string()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.number()
	case '-':
		d.pos++
		if d.pos >= d.end {
			return nil, ErrUnexpectedEOF
		}
		if c = d.data[d.pos]; c < '0' && c > '9' {
			return nil, d.error(c, "in negative numeric literal")
		}
		n, err := d.number()
		if err != nil {
			return nil, err
		}
		return -n, nil
	case '[':
		return d.array()
	case '{':
		return d.object()
	default:
		atom, err := d.atom()
		if err != nil {
			return nil, err
		}
		switch atom {
		case "true":
			return true, nil
		case "false":
			return false, nil
		case "null":
			return nil, nil
		case "int":
			return d.int()
		case "datetime":
			return d.datetime()
		case "ip":
			return d.ip()
		case "ipport":
			return d.ipport()
		case "int8":
			return d.int8()
		case "int16":
			return d.int16()
		case "int32":
			return d.int32()
		case "int64":
			return d.int64()
		case "uint":
			return d.uint()
		case "uint8":
			return d.uint8()
		case "uint16":
			return d.uint16()
		case "uint32":
			return d.uint32()
		case "uint64":
			return d.uint64()
		}
		return nil, d.error(c, "looking for beginning of value")
	}
}

func (d *Decoder) datetime() (time.Time, error) {
	str, err := d.bracketExpr()
	if err != nil {
		return time.Time{}, err
	}
	return time.Parse(time.RFC3339, str)
}

func (d *Decoder) ip() (net.IP, error) {
	str, err := d.bracketExpr()
	if err != nil {
		return nil, err
	}

	ip := net.ParseIP(str)
	if ip == nil {
		return nil, d.error(' ', "invalid ip")
	}

	return ip, nil
}

func (d *Decoder) ipport() (net.TCPAddr, error) {
	str, err := d.bracketExpr()
	if err != nil {
		return net.TCPAddr{}, err
	}

	if len(str) > 0 {
		var ipstr, portstr string
		var pos int
		if str[0] == '[' { // [ipv6]:port
			pos = strings.IndexByte(str[1:], ']')
			if pos == -1 {
				return net.TCPAddr{}, &SyntaxError{"invalid ipv6, missing ]", d.pos + 1}
			}
			pos++
			ipstr = str[1:pos]
			pos++
			if pos >= len(str) || str[pos] != ':' {
				return net.TCPAddr{}, &SyntaxError{"missing : after ipv6", d.pos + 1}
			}
		} else { // ipv4:port
			pos = strings.IndexByte(str, ':')
			if pos == -1 {
				return net.TCPAddr{}, &SyntaxError{"missing : after ipv4", d.pos + 1}
			}
			ipstr = str[:pos]
		}
		pos++
		if pos >= len(str) {
			return net.TCPAddr{}, &SyntaxError{"missing port after :", d.pos + 1}
		}
		portstr = str[pos:]
		ip := net.ParseIP(ipstr)
		if ip == nil {
			return net.TCPAddr{}, &SyntaxError{"malformed IP: " + ipstr, d.pos + 1}
		}
		port, err := strconv.Atoi(portstr)
		if err != nil {
			return net.TCPAddr{}, &SyntaxError{"malformed port: " + portstr, d.pos + 1}
		}
		return net.TCPAddr{IP: ip, Port: port}, nil
	}

	return net.TCPAddr{}, d.error(' ', "invalid ipport")
}

func (d *Decoder) uint() (uint, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(intStr, 10, 64)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return uint(n), nil
}

func (d *Decoder) uint8() (uint8, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(intStr, 10, 8)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return uint8(n), nil
}

func (d *Decoder) uint16() (uint16, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(intStr, 10, 16)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return uint16(n), nil
}

func (d *Decoder) uint32() (uint32, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(intStr, 10, 32)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return uint32(n), nil
}

func (d *Decoder) uint64() (uint64, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseUint(intStr, 10, 64)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return n, nil
}

func (d *Decoder) int() (int, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	num, err := strconv.Atoi(intStr)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return num, nil
}

func (d *Decoder) int8() (int8, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseInt(intStr, 10, 8)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return int8(n), nil
}

func (d *Decoder) int16() (int16, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseInt(intStr, 10, 16)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return int16(n), nil
}

func (d *Decoder) int32() (int32, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseInt(intStr, 10, 32)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return int32(n), nil
}

func (d *Decoder) int64() (int64, error) {
	intStr, err := d.bracketExpr()
	if err != nil {
		return 0, err
	}

	n, err := strconv.ParseInt(intStr, 10, 64)
	if err != nil {
		return 0, &SyntaxError{err.Error(), d.pos}
	}

	return n, nil
}

func (d *Decoder) objectKey() (string, error) {
	if d.pos >= d.end {
		return "", ErrUnexpectedEOF
	}
	if c := d.data[d.pos]; c == '"' {
		return d.string()
	} else {
		return d.atom()
	}
}

func (d *Decoder) atom() (string, error) {
	var c byte
	start := d.pos
	if d.pos < d.end {
		if c = d.data[d.pos]; c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' {
			d.pos++
			for d.pos < d.end {
				if c := d.data[d.pos]; c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '_' || c >= '0' && c <= '9' {
					d.pos++
				} else {
					break
				}
			}
			if d.usestring {
				return d.sdata[start:d.pos], nil
			}

			return string(d.data[start:d.pos]), nil
		}
	}

	return "", d.error(c, "looking for atom")
}

func (d *Decoder) bracketExpr() (string, error) {
	if c := d.skipSpaces(); c != '(' {
		return "", d.error(c, "looking for (")
	}

	d.pos++
	c := d.skipSpaces()
	start := d.pos
	if c == '"' {
		s, err := d.string()
		if err != nil {
			return "", err
		}
		if c := d.skipSpaces(); c != ')' {
			return "", d.error(c, "looking for )")
		}
		d.pos++
		return s, nil
	} else {
		for d.pos < d.end {
			if d.data[d.pos] == ')' {
				var ret string
				if d.usestring {
					ret = d.sdata[start:d.pos]
				} else {
					ret = string(d.data[start:d.pos])
				}
				d.pos++
				return ret, nil
			}
			d.pos++
		}
	}

	return "", d.error(' ', "looking for )")
}

// string called by `any` or `object`(for map keys) after reading `"`
func (d *Decoder) string() (string, error) {
	d.pos++

	var (
		unquote bool
		start   = d.pos
	)

scan:
	for {
		if d.pos >= d.end {
			return "", ErrUnexpectedEOF
		}

		c := d.data[d.pos]
		switch {
		case c == '"':
			var s string
			if unquote {
				// stack-allocated array for allocation-free unescaping of small strings
				// if a string longer than this needs to be escaped, it will result in a
				// heap allocation; idea comes from github.com/burger/jsonparser
				var stackbuf [64]byte
				data, ok := unquoteBytes(d.data[start:d.pos], stackbuf[:])
				if !ok {
					return "", ErrStringEscape
				}
				s = string(data)
			} else {
				if d.usestring {
					s = d.sdata[start:d.pos]
				} else {

					s = string(d.data[start:d.pos])
				}
			}
			d.pos++
			return s, nil
		case c == '\\':
			d.pos++
			if d.pos >= d.end {
				return "", ErrUnexpectedEOF
			}
			unquote = true
			switch c := d.data[d.pos]; c {
			case 'u':
				goto escape_u
			case 'b', 'f', 'n', 'r', 't', '\\', '/', '"':
				d.pos++
			default:
				return "", d.error(c, "in string escape code")
			}
		case c < 0x20:
			return "", d.error(c, "in string literal")
		default:
			d.pos++
			if c > unicode.MaxASCII {
				unquote = true
			}
		}
	}

escape_u:
	d.pos++
	for i := 0; i < 3; i++ {
		if d.pos < d.end {
			c := d.data[d.pos]
			if '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F' {
				d.pos++
				continue
			}
			return "", d.error(c, "in \\u hexadecimal character escape")
		}
		return "", ErrInvalidHexEscape
	}
	goto scan
}

// number called by `any` after reading number between 0 to 9
func (d *Decoder) number() (float64, error) {
	var (
		n       float64
		isFloat bool
		c       = d.data[d.pos]
		start   = d.pos
	)

	// digits first
	switch {
	case c == '0':
		c = d.next()
	case '1' <= c && c <= '9':
		for ; c >= '0' && c <= '9'; c = d.next() {
			n = 10*n + float64(c-'0')
		}
	}

	// . followed by 1 or more digits
	if c == '.' {
		d.pos++
		if d.pos >= d.end {
			return 0, ErrUnexpectedEOF
		}
		isFloat = true
		if c = d.data[d.pos]; c < '0' && c > '9' {
			return 0, d.error(c, "after decimal point in numeric literal")
		}
		for c = d.next(); '0' <= c && c <= '9'; {
			c = d.next()
		}
	}

	// e or E followed by an optional - or + and
	// 1 or more digits.
	if c == 'e' || c == 'E' {
		isFloat = true
		if c = d.next(); c == '+' || c == '-' {
			if c = d.next(); c < '0' || c > '9' {
				return 0, d.error(c, "in exponent of numeric literal")
			}
		}
		for c = d.next(); '0' <= c && c <= '9'; {
			c = d.next()
		}
	}

	if isFloat {
		var (
			err error
			sn  string
		)
		if d.usestring {
			sn = d.sdata[start:d.pos]
		} else {
			sn = string(d.data[start:d.pos])
		}
		if n, err = strconv.ParseFloat(sn, 64); err != nil {
			return 0, &SyntaxError{msg: err.Error(), Offset: d.pos}
		}
	}
	return n, nil
}

// array accept valid JSON array value
func (d *Decoder) array() ([]interface{}, error) {
	// the '[' token already scanned
	d.pos++

	var (
		c     byte
		v     interface{}
		err   error
		array = make([]interface{}, 0)
	)

scan:
	if c = d.skipSpaces(); c == ']' {
		d.pos++
		goto out
	}
	if v, err = d.any(); err != nil {
		goto out
	}

	array = append(array, v)

	// next token must be ',' or ']'
	if c = d.skipSpaces(); c == ',' {
		d.pos++
		goto scan
	} else if c == ']' {
		d.pos++
	} else {
		err = d.error(c, "after array element")
	}

out:
	return array, err
}

// object accept valid JSON array value
func (d *Decoder) object() (map[string]interface{}, error) {
	// the '{' token already scanned
	d.pos++

	var (
		c   byte
		k   string
		v   interface{}
		err error
		obj = make(map[string]interface{})
	)

	for {
		if c = d.skipSpaces(); c == '}' {
			d.pos++
			return obj, nil
		}

		// read key
		if k, err = d.objectKey(); err != nil {
			break
		}

		// read colon before value
		c = d.skipSpaces()
		if c != ':' {
			err = d.error(c, "after object key")
			break
		}
		d.pos++

		// read and assign value
		d.skipSpaces()
		if v, err = d.any(); err != nil {
			break
		}

		obj[k] = v

		// next token must be ',' or '}'
		if c = d.skipSpaces(); c == '}' {
			d.pos++
			break
		} else if c == ',' {
			d.pos++
		} else {
			err = d.error(c, "after object key:value pair")
			break
		}
	}

	return obj, err
}

// next return the next byte in the input
func (d *Decoder) next() byte {
	if d.pos < d.end {
		d.pos++
		if d.pos < d.end {
			return d.data[d.pos]
		}
	}
	return 0
}

// returns the next char after white spaces
func (d *Decoder) skipSpaces() byte {
loop:
	if d.pos == d.end {
		return 0
	}
	switch c := d.data[d.pos]; c {
	case ' ', '\t', '\n', '\r':
		d.pos++
		goto loop
	default:
		return c
	}
}

/*
for ;d.pos < d.end; d.pos++ {
		switch c := d.data[d.pos]; c {
		case ' ', '\t', '\n', '\r':
		default:
			return c
		}
	}

	return 0
}*/

// emit sytax errors
func (d *Decoder) error(c byte, context string) error {
	if d.pos < d.end {
		return &SyntaxError{"invalid character " + quoteChar(c) + " " + context, d.pos + 1}
	}
	return ErrUnexpectedEOF
}
