package tests

import "encoding/json"

// jsonMarshal is a test helper wrapping json.Marshal.
func jsonMarshal(v any) ([]byte, error) {
	return json.Marshal(v)
}
