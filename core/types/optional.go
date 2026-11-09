package types

import (
	"bytes"
	"encoding/json"
)

// Optional is a tri-state JSON value. Unlike a plain pointer, it distinguishes
// three cases for a field in an incoming JSON document:
//
//   - absent: the key is missing entirely → Set == false (leave stored value unchanged)
//   - explicit null: the key is present with value null → Set == true, Null == true (clear)
//   - value: the key is present with a concrete value → Set == true, Null == false, Value set
//
// This is required by PATCH/PUT contracts where a JSON null must be able to clear a
// nullable column back to its default, something a plain *T cannot express (nil pointer
// means both "absent" and "null").
//
// Because Optional is a struct, declarative `validate:` tags do NOT apply to its inner
// Value. Callers must validate the resolved value explicitly after unmarshaling.
type Optional[T any] struct {
	// Set is true when the JSON key was present (with any value, including null).
	Set bool
	// Null is true when the JSON key was present and its value was null.
	Null bool
	// Value holds the decoded value when Set && !Null.
	Value T
}

// UnmarshalJSON implements json.Unmarshaler. encoding/json only invokes it when the
// key is present, so a missing key leaves Set false. A present key sets Set; a literal
// null additionally sets Null; any other value is decoded into Value.
func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Set = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		o.Null = true
		var zero T
		o.Value = zero
		return nil
	}
	o.Null = false
	return json.Unmarshal(data, &o.Value)
}

// MarshalJSON implements json.Marshaler so an Optional round-trips in responses/Swagger
// examples: an absent or null Optional encodes as null, otherwise the inner value.
func (o Optional[T]) MarshalJSON() ([]byte, error) {
	if !o.Set || o.Null {
		return []byte("null"), nil
	}
	return json.Marshal(o.Value)
}

// IsSet reports whether the JSON key was present (with any value).
func (o Optional[T]) IsSet() bool { return o.Set }

// IsClear reports whether the JSON key was present and explicitly null (a request to
// clear the field).
func (o Optional[T]) IsClear() bool { return o.Set && o.Null }

// HasValue reports whether the JSON key carried a concrete (non-null) value.
func (o Optional[T]) HasValue() bool { return o.Set && !o.Null }
