// +build !tinygo

package vjson

import "encoding/json"

type RawMessage []byte

func Marshal(v interface{}) ([]byte, error) {

	// switch from vjson.RawMessage to json.RawMessage
	switch vt := v.(type) {
	case RawMessage:
		v = json.RawMessage(vt)
	}

	return json.Marshal(v)
}

func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
