// +build tinygo

package vjson

type RawMessage []byte

func Marshal(v interface{}) ([]byte, error) {
	panic("Marshal not yet implemented")
}

func Unmarshal(data []byte, v interface{}) error {
	panic("Unmarshal not yet implemented")
}
