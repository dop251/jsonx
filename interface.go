package jsonx

// A SyntaxError is a description of a JSON syntax error.
type SyntaxError struct {
	msg    string // description of error
	Offset int    // error occurred after reading Offset bytes
}

func (e *SyntaxError) Error() string { return e.msg }

// ExtraDataError is returned when a non-space data was found after parsing the top-level value.
// Offset contains the position of the first byte.
type ExtraDataError struct {
	Offset int
}

func (e *ExtraDataError) Error() string { return "Extra data after top-level value" }

// Predefined errors
var (
	ErrUnexpectedEOF    = &SyntaxError{"unexpected end of JSON input", -1}
	ErrInvalidHexEscape = &SyntaxError{"invalid hexadecimal escape sequence", -1}
	ErrStringEscape     = &SyntaxError{"encountered an invalid escape sequence in a string", -1}
)

// ValueType identifies the type of a parsed value.
type ValueType int

func (v ValueType) String() string {
	return types[v]
}

const (
	Null ValueType = iota
	Bool
	String
	Number
	Object
	Array
	Unknown
)

var types = map[ValueType]string{
	Null:    "null",
	Bool:    "boolean",
	String:  "string",
	Number:  "number",
	Object:  "object",
	Array:   "array",
	Unknown: "unknown",
}

// Type returns the JSON-type of the given value
func Type(v interface{}) ValueType {
	t := Unknown
	switch v.(type) {
	case nil:
		t = Null
	case bool:
		t = Bool
	case string:
		t = String
	case float64:
		t = Number
	case []interface{}:
		t = Array
	case map[string]interface{}:
		t = Object
	}
	return t
}

// Decode parses the JSON-encoded data and returns an interface value.
// Equivalent of NewDecoder(data).Decode()
func Decode(data []byte) (interface{}, error) {
	return NewDecoder(data).Decode()
}

// DecodeObject is the same as Decode but it returns map[string]interface{}.
// Equivalent of NewDecoder(data).DecodeObject()
func DecodeObject(data []byte) (map[string]interface{}, error) {
	return NewDecoder(data).DecodeObject()
}

// DecodeArray is the same as Decode but it returns []interface{}.
// Equivalent of NewDecoder(data).DecodeArray()
func DecodeArray(data []byte) ([]interface{}, error) {
	return NewDecoder(data).DecodeArray()
}
