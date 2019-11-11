// +build !tinygo

package vjson

import "encoding/json"

type RawMessage []byte

func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
