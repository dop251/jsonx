JSONX: extended JSON syntax for Go
==================================

JSONX is a superset of JSON that allows additional types and has a more
relaxed syntax.

Example:

```js
{
  k01: null,
  k02: false,
  k03: true,
  k04: "test",
  k05: 1.45678e-98,
  k06: int(-454365464),
  k07: uint(455645765),
  k08: int8(-128),
  k09: uint8(255),
  k10: int16(32767),
  k11: uint16(65535),
  k12: int32(2147483647),
  k13: uint32(4294967295),
  k14: int64("9223372036854775807"),
  k15: uint64("18446744073709551615"),
  k16: datetime("2017-12-25T15:00:00Z"),
  k17: ip("192.168.1.2"),
  k18: ipport("192.168.1.2:65000"),
  k19: ip("::1"),
  k20: ipport("[::1]:65000"),
  k21: [
    "test",
    int(123),
  ],
  k22: {
    test: true
  }
}
```

Differences from JSON:
----------------------

- Keys may be unquoted as long as they match ^\[A-Za-z_\]\[0-9A-Za-z_\]*$.
- Trailing commas after the last array or object elements are permitted.
- Additional types can be represented as 'type(value)'. The example above
  contains all currently supported types.

Usage
-----

The package includes a parser and a serialiser. They are both schemaless
(i.e. only accept and produce primitive values, \[\]interface{} and
map\[string\]interface{})

Because JSONX is a superset of JSON you can use the parser as a faster
alternative to the standard json.Unmarshal():

```go
var v interface{}
err := json.Unmarshal(data, &v)

// or
v, err := jsonx.Decode(data)

```

Non-greedy decoding example:

```go
b := []byte(`{test: 1} blah`)
v, err := jsonx.Decode(b)
if err, ok := err.(*jsonx.ExtraDataError); ok {
    tail := b[err.Offset:] // "blah"
    // parse tail
}
```

As JSONX is a valid ES5 expression you could parse it in Javascript
using eval() providing the type functions (int(), int8(), datetime(), etc..)
are defined.

Encoding example:

```go
v := map[string]interface{}{"test": true}

// encoding/json compatible API
b, err := jsonx.Marshal(v)
b1, err := jsonx.MarshalIndent(v, ">", "\t")

// encoding to an io.Writer
enc := jsonx.NewEncoder(os.Stdout)
err = enc.Encode(v)
enc1 := jsonx.NewEncoderIndent(os.Stdout, ">", "\t")
err = enc1.Encode(v)
```

Acknowledgements
----------------

Decoder is based on [djson](https://github.com/a8m/djson) which saved me
a lot of time writing it from scratch.

The code was tested with [go-fuzz](https://github.com/dvyukov/go-fuzz) which
I think is a must for any projects like this.
