package utils

import (
	"testing"
)

func TestMaskJSONString(t *testing.T) {
	tests := []struct {
		name          string
		jsonStr       string
		sensitiveKeys []string
		expected      string
	}{
		{
			name:          "no sensitive keys",
			jsonStr:       `{"name":"test","key":"secret"}`,
			sensitiveKeys: nil,
			expected:      `{"name":"test","key":"secret"}`,
		},
		{
			name:          "mask single key",
			jsonStr:       `{"name":"test","api_key":"super-secret-123"}`,
			sensitiveKeys: []string{"api_key"},
			expected:      `{"api_key":"***","name":"test"}`,
		},
		{
			name:          "mask multiple keys",
			jsonStr:       `{"name":"test","api_key":"secret1","token":"secret2"}`,
			sensitiveKeys: []string{"api_key", "token"},
			expected:      `{"api_key":"***","name":"test","token":"***"}`,
		},
		{
			name:          "mask nested key",
			jsonStr:       `{"input":{"api_key":"nested-secret","name":"test"}}`,
			sensitiveKeys: []string{"api_key"},
			expected:      `{"input":{"api_key":"***","name":"test"}}`,
		},
		{
			name:          "mask in array",
			jsonStr:       `{"items":[{"api_key":"secret1"},{"api_key":"secret2"}]}`,
			sensitiveKeys: []string{"api_key"},
			expected:      `{"items":[{"api_key":"***"},{"api_key":"***"}]}`,
		},
		{
			name:          "invalid json returns original",
			jsonStr:       `not valid json`,
			sensitiveKeys: []string{"key"},
			expected:      `not valid json`,
		},
		{
			name:          "key not present",
			jsonStr:       `{"name":"test"}`,
			sensitiveKeys: []string{"api_key"},
			expected:      `{"name":"test"}`,
		},
		{
			name:          "empty sensitive keys",
			jsonStr:       `{"api_key":"secret"}`,
			sensitiveKeys: []string{},
			expected:      `{"api_key":"secret"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskJSONString(tt.jsonStr, tt.sensitiveKeys)
			if result != tt.expected {
				t.Errorf("MaskJSONString() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestMaskSensitiveValues(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		sensitiveInputs interface{}
		expected        string
	}{
		{
			name:            "nil sensitive inputs",
			input:           "error: secret-value leaked",
			sensitiveInputs: nil,
			expected:        "error: secret-value leaked",
		},
		{
			name:  "mask value in string",
			input: "error: authentication failed with token my-secret-token",
			sensitiveInputs: map[string]interface{}{
				"token": "my-secret-token",
			},
			expected: "error: authentication failed with token ***",
		},
		{
			name:  "mask multiple values",
			input: "connecting to https://api.example.com with key abc123",
			sensitiveInputs: map[string]interface{}{
				"url": "https://api.example.com",
				"key": "abc123",
			},
			expected: "connecting to *** with key ***",
		},
		{
			name:  "no match leaves string unchanged",
			input: "error: something went wrong",
			sensitiveInputs: map[string]interface{}{
				"token": "not-in-string",
			},
			expected: "error: something went wrong",
		},
		{
			name:  "empty string value is not masked",
			input: "error: something went wrong",
			sensitiveInputs: map[string]interface{}{
				"empty": "",
			},
			expected: "error: something went wrong",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSensitiveValues(tt.input, tt.sensitiveInputs)
			if result != tt.expected {
				t.Errorf("MaskSensitiveValues() = %q, want %q", result, tt.expected)
			}
		})
	}
}
