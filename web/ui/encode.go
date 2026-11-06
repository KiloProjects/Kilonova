package ui

import (
	"encoding/base64"
	"encoding/json"
)

// EncodeJSON is v as base64 JSON, the format the Preact components read from their enc attribute.
func EncodeJSON(v any) string {
	b, _ := json.Marshal(v) // only used on plain data structs, which always marshal
	return base64.StdEncoding.EncodeToString(b)
}

// JS is v as a JavaScript literal, safe to splice into an inline <script>:
// json.Marshal escapes <, > and &, so the value can't close the script tag.
func JS(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
