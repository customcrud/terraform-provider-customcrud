package utils

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

// MaskInputKeys returns the payload JSON with the values of the given top-level
// input keys replaced by "***". Nothing else in the payload is altered.
func MaskInputKeys(payload []byte, keys []string) string {
	if len(keys) == 0 {
		return string(payload)
	}
	var data map[string]interface{}
	d := json.NewDecoder(bytes.NewReader(payload))
	d.UseNumber()
	if err := d.Decode(&data); err != nil {
		return string(payload)
	}
	input, ok := data["input"].(map[string]interface{})
	if !ok {
		return string(payload)
	}
	for _, k := range keys {
		if _, ok := input[k]; ok {
			input[k] = "***"
		}
	}
	masked, err := json.Marshal(data)
	if err != nil {
		return string(payload)
	}
	return string(masked)
}

// SensitiveValues returns the non-empty string values of the given top-level input keys.
func SensitiveValues(input interface{}, keys []string) []string {
	m, ok := input.(map[string]interface{})
	if !ok {
		return nil
	}
	var values []string
	for _, k := range keys {
		if s, ok := m[k].(string); ok && s != "" {
			values = append(values, s)
		}
	}
	return values
}

// MaskValues replaces exact occurrences of each value in s with "***".
func MaskValues(s string, values []string) string {
	sort.Slice(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	for _, v := range values {
		s = strings.ReplaceAll(s, v, "***")
	}
	return s
}
