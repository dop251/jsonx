package jsonx

import (
	"fmt"
	"math"
	"net"
	"testing"
	"time"
)

var (
	testMap = map[string]interface{}{
		"k01": nil,
		"k02": false,
		"k03": true,
		"k04": "test",
		"k05": 1.45678e-98,
		"k06": int(-454365464),
		"k07": uint(455645765),
		"k08": int8(-128),
		"k09": uint8(255),
		"k10": int16(32767),
		"k11": uint16(65535),
		"k12": int32(math.MaxInt32),
		"k13": uint32(math.MaxUint32),
		"k14": int64(math.MaxInt64),
		"k15": uint64(math.MaxUint64),
		"k16": time.Date(2017, 12, 25, 15, 0, 0, 0, time.UTC),
		"k17": net.IPv4(192, 168, 1, 2),
		"k18": net.TCPAddr{IP: net.IPv4(192, 168, 1, 2), Port: 65000},
		"k19": net.IPv6loopback,
		"k20": net.TCPAddr{IP: net.IPv6loopback, Port: 65000},
		"k21": []interface{}{"test", 123},
		"k22": map[string]interface{}{
			"test": true,
		},
	}
)

func TestMarshal(t *testing.T) {
	b, err := Marshal(testMap)
	if err != nil {
		t.Fatal(err)
	}

	if s := string(b); s != `{k01:null,k02:false,k03:true,k04:"test",k05:1.45678e-98,k06:int(-454365464),k07:uint(455645765),k08:int8(-128),k09:uint8(255),k10:int16(32767),k11:uint16(65535),k12:int32(2147483647),k13:uint32(4294967295),k14:int64("9223372036854775807"),k15:uint64("18446744073709551615"),k16:datetime("2017-12-25T15:00:00Z"),k17:ip("192.168.1.2"),k18:ipport("192.168.1.2:65000"),k19:ip("::1"),k20:ipport("[::1]:65000"),k21:["test",int(123)],k22:{test:true}}` {
		t.Fatalf("Unexpected value: '%s'", s)
	}
}

func TestMarshalIndent(t *testing.T) {
	b, err := MarshalIndent(testMap, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	if s := string(b); s != `{
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
    int(123)
  ],
  k22: {
    test: true
  }
}` {
		fmt.Print(s)
		t.Fatalf("Unexpected value: '%s'", s)
	}
}
