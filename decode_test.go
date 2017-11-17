package jsonx

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

type decodeTest struct {
	in       string
	err      error
	expected interface{}
}

var decodeTests = []decodeTest{
	// basic types
	{in: "null", expected: nil},
	{in: `true`, expected: true},
	{in: `false`, expected: false},
	{in: `5`, expected: 5.0},
	{in: `-5`, expected: -5.0},
	{in: `5.5`, expected: 5.5},
	{in: `"a\u1234"`, expected: "a\u1234"},
	{in: `"http:\/\/"`, expected: "http://"},
	{in: `"g-clef: \uD834\uDD1E"`, expected: "g-clef: \U0001D11E"},
	{in: `"invalid: \uD834x\uDD1E"`, expected: "invalid: \uFFFDx\uFFFD"},
	{in: `{"X": [1], "Y": 4}`, expected: map[string]interface{}{
		"X": []interface{}{1.0},
		"Y": 4.0,
	}},
	{in: `{"k1":1e-3,"k2":"s","k3":[1,2.0,3e-3],"k4":{"kk1":"s","kk2":2}}`, expected: map[string]interface{}{
		"k1": 1e-3,
		"k2": "s",
		"k3": []interface{}{1.0, 2.0, 3e-3},
		"k4": map[string]interface{}{
			"kk1": "s",
			"kk2": 2.0,
		},
	}},

	// extended syntax
	{in: `{k1: 1e-3, k2: int(64), k3: int64(444444444444442), k4: datetime("2017-01-01T12:00:00Z"),
	k5: ip("192.168.100.19"), k6: ip("fd00::abc:1"), k7: ipport("192.168.100.001:65555"),
	k8: ipport("[fd00::abc:1]:65555"),
	k9: int8(123),
	k10: int8(-123),
	k11: int16(4567),
	k12: int16(-4567),
	k13: int32(5364564),
	k14: int32(-4656453),
	k15: int64("9223372036854775807"),
	__: true,
	_a_b : false ,
	}`, expected: map[string]interface{}{
		"k1":   1e-3,
		"k2":   int(64),
		"k3":   int64(444444444444442),
		"k4":   time.Date(2017, 1, 1, 12, 0, 0, 0, time.UTC),
		"k5":   net.IPv4(192, 168, 100, 19),
		"k6":   net.ParseIP("fd00::abc:1"),
		"k7":   net.TCPAddr{IP: net.IPv4(192, 168, 100, 1), Port: 65555},
		"k8":   net.TCPAddr{IP: net.ParseIP("fd00::abc:1"), Port: 65555},
		"k9":   int8(123),
		"k10":  int8(-123),
		"k11":  int16(4567),
		"k12":  int16(-4567),
		"k13":  int32(5364564),
		"k14":  int32(-4656453),
		"k15":  int64(9223372036854775807),
		"__":   true,
		"_a_b": false,
	}},
	{in: `["test", int64(-123),]`, expected: []interface{}{
		"test",
		int64(-123),
	}},
	{in: `[,]`, err: &SyntaxError{"invalid character ',' looking for atom", 2}},

	// int range error
	{in: `int8(-500)`, err: &SyntaxError{"strconv.ParseInt: parsing \"-500\": value out of range", 10}},

	// raw values with whitespace
	{in: "\n true ", expected: true},
	{in: "\n false ", expected: false},
	{in: "\t 1 ", expected: float64(1)},
	{in: "\r 1.2 ", expected: 1.2},
	{in: "\t -5 \n", expected: float64(-5)},
	{in: "\t \"a\\u1234\" \n", expected: "a\u1234"},

	// EOF errors
	{in: `fal`, err: ErrUnexpectedEOF},
	{in: `nul`, err: ErrUnexpectedEOF},
	{in: `tru`, err: ErrUnexpectedEOF},
	{in: `"as`, err: ErrUnexpectedEOF},
	{in: `[1,`, err: ErrUnexpectedEOF},

	// syntax errors
	{in: `{"X": "foo", "Y"}`, err: &SyntaxError{"invalid character '}' after object key", 17}},
	{in: `[1, 2, 3+]`, err: &SyntaxError{"invalid character '+' after array element", 9}},
	{in: `{"X":12x}`, err: &SyntaxError{"invalid character 'x' after object key:value pair", 8}},
	{in: `int(failme)`, err: &SyntaxError{"strconv.Atoi: parsing \"failme\": invalid syntax", 11}},

	// raw value errors
	{in: "\x01 42", err: &SyntaxError{"invalid character '\\x01' looking for atom", 1}},
	{in: " 42 \x01", expected: 42.0, err: &ExtraDataError{4}},
	{in: "\x01 true", err: &SyntaxError{"invalid character '\\x01' looking for atom", 1}},
	{in: " false \x01", expected: false, err: &ExtraDataError{7}},
	{in: "\x01 1.2", err: &SyntaxError{"invalid character '\\x01' looking for atom", 1}},
	{in: " 3.4 \x01", expected: 3.4, err: &ExtraDataError{5}},
	{in: "\x01 \"string\"", err: &SyntaxError{"invalid character '\\x01' looking for atom", 1}},
	{in: " \"string\" \x01", expected: "string", err: &ExtraDataError{10}},

	// array tests
	{in: `[1, 2, 3]`, expected: []interface{}{1.0, 2.0, 3.0}},
	{in: `{"T":[1]}`, expected: map[string]interface{}{
		"T": []interface{}{1.0},
	}},
	{in: `{"T":null}`, expected: map[string]interface{}{
		"T": nil,
	}},

	// invalid UTF-8 is coerced to valid UTF-8.
	{in: "\"hello\xffworld\"", expected: "hello\ufffdworld"},
	{in: "\"hello\xc2\xc2world\"", expected: "hello\ufffd\ufffdworld"},
	{in: "\"hello\xc2\xffworld\"", expected: "hello\ufffd\ufffdworld"},
	{in: "\"hello\\ud800world\"", expected: "hello\ufffdworld"},
	{in: "\"hello\\ud800\\ud800world\"", expected: "hello\ufffd\ufffdworld"},
	{in: "\"hello\\ud800\\ud800world\"", expected: "hello\ufffd\ufffdworld"},
	{in: "\"hello\xed\xa0\x80\xed\xb0\x80world\"", expected: "hello\ufffd\ufffd\ufffd\ufffd\ufffd\ufffdworld"},

	// crashes
	{in: `["\`, err: ErrUnexpectedEOF},
	{in: `0.`, err: ErrUnexpectedEOF},
	{in: `-`, err: ErrUnexpectedEOF},
	{in: `{`, err: ErrUnexpectedEOF},
	{in: `"\u0`, err: ErrInvalidHexEscape},
	{in: `0e`, err: &SyntaxError{"strconv.ParseFloat: parsing \"0e\": invalid syntax", 2}},
}

func TestDecode(t *testing.T) {
	for i, tt := range decodeTests {
		tt := tt
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			t.Parallel()
			b := []byte(tt.in)
			b = b[:len(b):len(b)] // prevent access beyond the length
			out, err := Decode(b)
			if !reflect.DeepEqual(err, tt.err) {
				t.Errorf("%v (%T), want %v", err, err, tt.err)
			}
			if out != nil {
				if !reflect.DeepEqual(out, tt.expected) {
					t.Errorf("%v, want %v", out, tt.expected)
				}
			}
		})
	}
}

var allValueIndent = []byte(`{
	"null_1": null,
	"null_2":     null,
	"null_3":null       ,

	"bool_1": false,
	"bool_2": true,
	"bool_3":         false,
	"bool_4":true        ,

	"string_1": "This is a string",
	"string_2": "Déjà vu",
	"string_3": "",
	"string_4": "null",
	"string_5": "5",
	"string_6": "\"foobar\"<html> [\u2028 \u2029]",

	"int_1": 42,
	"int_2": -1,
	"int_3": 11111111,
	"float_1": 3.1415926,
	"float_2": -0.1415926,
	"float_3": 0.1415926,
	"float_4": 2.99792458e8,
	"float_5": 7.71234e+1,
	"float_6": 1.234e-1,
	"float_7": 1.234e-1      ,
	"float_8":     1.234e-1,

	"array_1": [],
	"array_2": [2,3,4,4],
	"array_3": [2, "a", "3", "v", true, false],
	"array_4": [[], ["a", "d"], [[[false]]]],
	"array_5": [{"a": 1}, {"d": 2}, "d", "d", "s", "a", 3, 3],
	"array_5": [{"a": 1}, {"d": 2}, "d", "d", "s", "a", 3, {
		"array_5_1": [
			{
				"array_5_1_1": ["a", "b", "c", "d"],
				"array_5_1_2": [1,2,2,3,4,5,5,6,0,7,7]
			}
		]
	}],

	"object_1": {},
	"object_2": {"a": 1},
	"object_3": {"a": 1, "b": 3},
	"object_4": {"a": 1, "c": { "d"     : "d", "f":            2}},
	"object_5": {"a": 1, "s": "{}", "ss": "{\"ss\": 2}" },
	"object_6": {
		"a": 2,
		"b": "a",
		"c": {
			"ca": 2,
			"cb": "a",
			"cc": {
				"cca": [1,2,3,4,5,6,true, false, "a"],
				"ccb": [],
				"ccc": "[]",
				"ccd": {
					"ccda": [
						1,
						2,
						3,
						4,
						5,
						6
					]
				}
			}
		}
	}
}`)

func TestWithStdDecoder(t *testing.T) {
	expected := make(map[string]interface{})
	if err := json.Unmarshal(allValueIndent, &expected); err != nil {
		t.Errorf("expecting std json not to fail: %q", err)
	}
	out, err := Decode(allValueIndent)
	if err != nil {
		t.Errorf("expecting decode not to fail: %q", err)
	}
	if actual := out.(map[string]interface{}); !reflect.DeepEqual(actual, expected) {
		t.Errorf("compare to std unmarshaler \n\tactual: %v\n\twant: %v", actual, expected)

	}
}

func TestDecodeArray(t *testing.T) {
	for i, tt := range []struct {
		err      error
		in       string
		expected []interface{}
	}{
		{in: `["a"]`, expected: []interface{}{"a"}},
		{in: `["a"]   `, expected: []interface{}{"a"}},
		{in: `   ["a"]`, expected: []interface{}{"a"}},
		{in: `   [     "a"]`, expected: []interface{}{"a"}},
		{in: `   ["a"      ]`, expected: []interface{}{"a"}},
		{in: `["a"      ]1`, expected: []interface{}{"a"}, err: &ExtraDataError{11}},
	} {
		out, err := DecodeArray([]byte(tt.in))
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("#%d: %v, want %v", i, err, tt.err)
		}
		if !reflect.DeepEqual(out, tt.expected) {
			t.Errorf("#%d: %v, want %v", i, out, tt.expected)
		}
	}
}

func TestDecodeObject(t *testing.T) {
	for i, tt := range []struct {
		err      error
		in       string
		expected map[string]interface{}
	}{
		{in: `{"a":"a"}`, expected: map[string]interface{}{"a": "a"}},
		{in: `   {"a":"a"}`, expected: map[string]interface{}{"a": "a"}},
		{in: `{"a":"1"}   `, expected: map[string]interface{}{"a": "1"}},
		{in: `{   "a":"1"}`, expected: map[string]interface{}{"a": "1"}},
		{in: `{"a"   :1  }`, expected: map[string]interface{}{"a": float64(1)}},
		{in: `{"a":1}   1`, expected: map[string]interface{}{"a": float64(1)}, err: &ExtraDataError{10}},
	} {
		out, err := DecodeObject([]byte(tt.in))
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("#%d: %v, want %v", i, err, tt.err)
		}
		if !reflect.DeepEqual(out, tt.expected) {
			t.Errorf("#%d: %v, want %v", i, out, tt.expected)
		}
	}
}

func TestType(t *testing.T) {
	var tests = map[ValueType]string{
		Null:   "null",
		Bool:   "true",
		String: "\"string\"",
		Number: "123",
		Object: "{}",
		Array:  "[]",
	}
	for k, v := range tests {
		k, v := k, v
		t.Run(k.String(), func(t *testing.T) {
			t.Parallel()
			vv, _ := Decode([]byte(v))
			if vt := Type(vv); vt != k {
				t.Errorf("Type(%s) = %q; want %q", k, vt, k)
			}
		})
	}
}

func TestFuzzCover(t *testing.T) {
	d, err := os.Open("testdata/fuzz/corpus")
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip(err)
		}
		t.Fatal(err)
	}

	files, err := d.Readdir(-1)
	if err != nil {
		t.Fatal(err)
	}

	for _, file := range files {
		if file.Mode().IsRegular() {
			f, err := os.Open(filepath.Join(d.Name(), file.Name()))
			if err != nil {
				t.Fatal(err)
			}

			data, err := ioutil.ReadAll(f)
			if err != nil {
				t.Fatal(err)
			}

			f.Close()
			decoded, err := Decode(data)
			if err == nil {
				Marshal(decoded)
			}
		}
	}
}

func TestNonGreedy(t *testing.T) {
	b := []byte(`{test: 1} blah`)
	v, err := Decode(b)
	if err1, ok := err.(*ExtraDataError); ok {
		tail := b[err1.Offset:]
		if string(tail) != "blah" {
			t.Fatalf("Unexpected tail: '%s'", tail)
		}
		expected := map[string]interface{}{
			"test": 1.0,
		}
		if !reflect.DeepEqual(v, expected) {
			t.Fatalf("Unexpected value: %v", v)
		}
	} else {
		t.Fatalf("Unexpected error: %v", err)
	}

}

func BenchmarkDecode(b *testing.B) {
	data := []byte(`{k1: 1e-3, k2: int(64), k3: int64(444444444444442), k4: datetime("2017-01-01T12:00:00Z"),
	k5: ip("192.168.100.19"), k6: ip("fd00::abc:1"), k7: ipport("192.168.100.001:65555"),
	k8: ipport("[fd00::abc:1]:65555")}`)
	for i := 0; i < b.N; i++ {
		Decode(data)
	}
}
