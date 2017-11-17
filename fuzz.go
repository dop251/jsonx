// +build gofuzz

package jsonx

func Fuzz(data []byte) int {
	v, err := Decode(data)
	if err != nil {
		return 0
	}

	_, err = Marshal(v)

	return 1
}
