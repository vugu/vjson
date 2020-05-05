// Package vjson is a small, minimal alternative to encoding/json which has no dependencies
// and works with Tingyo.  The Marshal and Unmarshal methods work like encoding/json, but
// structs are not supported, only primitives plus map[string]interface{} and []interface{}.
package vjson

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// NOTE: various bits have been borrowed from encoding/json

// bool, for JSON booleans
// float64, for JSON numbers
// string, for JSON strings
// []interface{}, for JSON arrays
// map[string]interface{}, for JSON objects
// nil for JSON null
// RawMessage - just use Marshaler

// if someone asks to ask into an int type that should still work

func marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := marshalTo(&buf, v)
	return buf.Bytes(), err
}

func marshalTo(w io.Writer, vin interface{}) (err error) {

	bb := make([]byte, 0, 64) // hopefully stack alloc

	// TODO: check for Marshaler

	// TODO: check for MarshalerTo

	if vin == nil {
		_, err = w.Write([]byte("null"))
		return err
	}

	switch v := vin.(type) {

	case bool:
		bb = strconv.AppendBool(bb, v)

	case float64:
		return float64Encoder(w, v)

	case float32:
		return float32Encoder(w, float64(v))

	case int:
		bb = strconv.AppendInt(bb, int64(v), 10)
	case int8:
		bb = strconv.AppendInt(bb, int64(v), 10)
	case int16:
		bb = strconv.AppendInt(bb, int64(v), 10)
	case int32:
		bb = strconv.AppendInt(bb, int64(v), 10)
	case int64:
		bb = strconv.AppendInt(bb, v, 10)
	case uint:
		bb = strconv.AppendUint(bb, uint64(v), 10)
	case uint8:
		bb = strconv.AppendUint(bb, uint64(v), 10)
	case uint16:
		bb = strconv.AppendUint(bb, uint64(v), 10)
	case uint32:
		bb = strconv.AppendUint(bb, uint64(v), 10)
	case uint64:
		bb = strconv.AppendUint(bb, v, 10)

	case string:
		return encodeString(w, v, false)

	// case []byte: // TODO: this is wrong - byte slice should get base64 encoded
	// 	return encodeStringBytes(w, v, false)

	case []interface{}:
		if v == nil {
			bb = append(bb, `null`...)
			break
		}
		w.Write([]byte(`[`))
		first := true
		for i := range v {
			if !first {
				w.Write([]byte(`,`))
			}
			first = false
			err := marshalTo(w, v[i])
			if err != nil {
				return err
			}
		}
		_, err := w.Write([]byte(`]`))
		return err

	case map[string]interface{}:
		if v == nil {
			bb = append(bb, `null`...)
			break
		}
		w.Write([]byte(`{`))
		first := true
		for k, el := range v {
			if !first {
				w.Write([]byte(`,`))
			}
			first = false
			encodeString(w, k, false)
			w.Write([]byte(`:`))
			err := marshalTo(w, el)
			if err != nil {
				return err
			}
		}
		_, err := w.Write([]byte(`}`))
		return err

	// case interface{}: // needed?

	// TODO: pointer cases

	default:
		return fmt.Errorf("vjson.marshalTo error unknown type: %T", vin)
	}

	if len(bb) == 0 {
		panic("unexpected zero length buffer") // should never happen
	}

	_, err = w.Write(bb)
	return err
}

func unmarshal(data []byte, v interface{}) error {
	return unmarshalFrom(bytes.NewReader(data), v)
}

func unmarshalFrom(r unreader, vin interface{}) error {

	// read the next token, whatever it is
	tok, err := readToken(r)
	if err != nil {
		return err
	}

	return unmarshalNext(r, tok, vin)
}

func unmarshalNext(r unreader, tok Token, vin interface{}) error {

	tokDelim, _ := tok.(Delim)

	// and type switch to determine how to handle it
	switch v := vin.(type) {

	case *interface{}:
		if tok == nil {
			*v = nil
		} else {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

	case *bool:
		if tokv, ok := tok.(bool); ok {
			*v = tokv
		} else {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

	case *float64:
		if tokv, ok := tok.(Number); ok {
			f, err := tokv.Float64()
			if err != nil {
				return err
			}
			*v = f
		} else {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

	// case *float32:

	// case *int:

	// case *int8:

	// case *int16:

	// case *int32:

	// case *int64:

	// case *uint:

	// case *uint8:

	// case *uint16:

	// case *uint32:

	// case *uint64:

	case *string:
		if tokv, ok := tok.(string); ok {
			*v = tokv
		} else {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

	// case *[]byte: // hm, should be base64 encoded
	// 	if tokv, ok := tok.(string); ok {
	// 		*v = []byte(tokv)
	// 	} else {
	// 		return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
	// 	}

	case *[]interface{}:

		// make sure we have an array start
		if tokDelim != Delim('[') {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

		sliceV := make([]interface{}, 0, 4)

		for {
			nextTok, err := readToken(r)
			if err != nil {
				return err
			}
			nextTokDelim, _ := nextTok.(Delim)
			if nextTokDelim == Delim(']') {
				break // end array
			}
			elV := newDefaultForToken(nextTok)
			err = unmarshalNext(r, nextTok, elV)
			if err != nil {
				return err
			}
			sliceV = append(sliceV, deref(elV))
		}

		*v = sliceV

	case *map[string]interface{}:

		// make sure we have an object start
		if tokDelim != Delim('{') {
			return fmt.Errorf("vjson.unmarshalNext unable to scan %#v into %T", tok, v)
		}

		mapV := make(map[string]interface{}, 4)

		for {
			// read object key (must be string)
			keyTok, err := readToken(r)
			if err != nil {
				return err
			}
			keyTokDelim, _ := keyTok.(Delim)
			if keyTokDelim == Delim('}') {
				break // end object
			}

			nextTok, err := readToken(r)
			if err != nil {
				return err
			}

			elV := newDefaultForToken(nextTok)
			err = unmarshalNext(r, nextTok, elV)
			if err != nil {
				return err
			}

			keyTokStr, ok := keyTok.(string)
			if !ok {
				return fmt.Errorf("unexpected non-string object key token: %#v", keyTok)
			}
			mapV[keyTokStr] = deref(elV)

		}

		*v = mapV

	default:
		return fmt.Errorf("vjson.unmarshalNext error unknown type: %T", vin)

	}

	return nil
}

// newDefaultForToken will return a pointer to the appropriate type based on a JSON token.
// Used when scanning into an interface{} and we need to infer the Go type from the JSON input.
// A nil Token will return nil.
func newDefaultForToken(tok Token) interface{} {

	if tok == nil {
		return new(interface{})
	}

	switch tok.(type) {
	case bool:
		return new(bool)
	case Number:
		return new(float64)
	case float64:
		return new(float64)
	case string:
		return new(string)
	}

	tokDelim, _ := tok.(Delim)
	if tokDelim == Delim('[') {
		return new([]interface{})
	} else if tokDelim == Delim('{') {
		return new(map[string]interface{})
	}

	panic(fmt.Errorf("newDefaultForToken unexpected token %v (type=%T)", tok, tok))
}

// deref will strip the pointer off of the value returned by newDefaultForToken
func deref(vin interface{}) interface{} {

	// if vin == nil {
	// 	return nil
	// }

	switch v := vin.(type) {
	case *interface{}:
		if *v == nil {
			return nil
		} else {
			panic(fmt.Errorf("deref: *interface{} should have been nil but got: %+v", *v))
		}
	case *bool:
		return *v
	case *string:
		return *v
	case *float64:
		return *v
	case *[]interface{}:
		return *v
	case *map[string]interface{}:
		return *v
	}

	panic(fmt.Errorf("vjson.deref got unknown type %T", vin))
}

// unreader is implemented by bytes.Reader and bytes.Buffer
type unreader interface {
	Read(p []byte) (n int, err error)
	// ReadBytes(delim byte) (line []byte, err error)
	ReadByte() (byte, error)
	UnreadByte() error
}

// NOTE: for writing io.Writer works, but for reading io.Reader does NOT work because
// not all JSON data types have a termination character (e.g. you cannot tell when you've
// reached the end of a number without reading past it).  One solution could be to define
// an interface with the methods we need from bytes.Reader, minimally Read() and UnreadByte()

// type MarshalerTo interface {
// 	MarshalJSONTo(w io.Writer) error
// }

// Marshaler is the interface implemented by types that can marshal themselves into valid JSON.
type Marshaler interface {
	MarshalJSON() ([]byte, error)
}

// type UnmarshalerFrom interface {
// 	UnmarshalJSONFrom(r io.Reader) error
// }

// Unmarshaler is the interface implemented by types that can unmarshal a JSON description of themselves.
type Unmarshaler interface {
	UnmarshalJSON([]byte) error
}

// RawMessage is a raw encoded JSON value.
// It implements Marshaler and Unmarshaler and can
// be used to delay JSON decoding or precompute a JSON encoding.
type RawMessage []byte

// MarshalJSON returns m as the JSON encoding of m.
func (m RawMessage) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	return m, nil
}

// UnmarshalJSON sets *m to a copy of data.
func (m *RawMessage) UnmarshalJSON(data []byte) error {
	if m == nil {
		return errors.New("vjson.RawMessage: UnmarshalJSON on nil pointer")
	}
	*m = append((*m)[0:0], data...)
	return nil
}

var _ Marshaler = (*RawMessage)(nil)
var _ Unmarshaler = (*RawMessage)(nil)
