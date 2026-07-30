package hrobot

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// StringList holds a Robot field that arrives either as one string or as an
// array of strings, and presents both as a slice.
//
// Boot configuration is the case this exists for. While the rescue system is
// active, "os" is the string naming the running system. While it is inactive,
// "os" is the array of systems that could be booted. A one-element list
// therefore means that value is the active one. The same applies to the "dist"
// and "lang" fields, and to a cancellation's "cancellation_reason".
//
// A JSON null decodes to a nil list.
type StringList []string

// UnmarshalJSON implements [json.Unmarshaler], accepting a bare string, an
// array of strings, or null.
func (l *StringList) UnmarshalJSON(data []byte) error {
	return decodeScalarOrArray(data, (*[]string)(l))
}

// IntList holds a Robot field that arrives either as one integer or as an array
// of integers, and presents both as a slice. See [StringList] for why the API
// varies the shape.
//
// The only current use is the boot "arch" field, which Hetzner has marked
// deprecated.
//
// A JSON null decodes to a nil list.
type IntList []int

// UnmarshalJSON implements [json.Unmarshaler], accepting a bare integer, an
// array of integers, or null.
func (l *IntList) UnmarshalJSON(data []byte) error {
	return decodeScalarOrArray(data, (*[]int)(l))
}

// decodeScalarOrArray decodes either a bare T or an array of T into dst.
//
// It dispatches on the first meaningful byte rather than attempting the array
// decode and falling back on failure, so a genuinely malformed array reports
// its own parse error instead of being retried as a scalar and reported as the
// wrong type.
func decodeScalarOrArray[T any](data []byte, dst *[]T) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		*dst = nil
		return nil
	}

	if trimmed[0] == '[' {
		var many []T
		if err := json.Unmarshal(trimmed, &many); err != nil {
			return fmt.Errorf("decode array: %w", err)
		}
		*dst = many
		return nil
	}

	var one T
	if err := json.Unmarshal(trimmed, &one); err != nil {
		return fmt.Errorf("decode scalar: %w", err)
	}
	*dst = []T{one}
	return nil
}
