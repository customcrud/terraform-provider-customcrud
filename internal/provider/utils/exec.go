package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ExecutionPayload struct {
	Id     string      `json:"id,omitempty"`
	Input  interface{} `json:"input,omitempty"`
	Output interface{} `json:"output,omitempty"`
}

type ExecutionResult struct {
	Payload       string
	Result        map[string]interface{}
	Stdout        string
	Stderr        string
	ExitCode      int
	MaskedPayload string
	MaskedStdout  string
	MaskedStderr  string
}

// Execute runs the given command with the provided payload, returning the result and any error.
func Execute(ctx context.Context, config CustomCRUDProviderConfig, cmd []string, payload ExecutionPayload) (*ExecutionResult, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	payloadStr := string(payloadBytes)
	maskedPayload := MaskJSONString(payloadStr, config.SensitiveKeys)
	tflog.Debug(ctx, "Executing script", map[string]interface{}{
		"command": cmd,
		"payload": maskedPayload,
	})

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Stdin = bytes.NewReader(payloadBytes)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err = execCmd.Run()
	result := &ExecutionResult{
		Payload:  payloadStr,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}
	result.MaskedPayload = maskedPayload
	result.MaskedStdout = MaskJSONString(result.Stdout, config.SensitiveKeys)
	result.MaskedStderr = MaskSensitiveValues(result.Stderr, config.SensitiveDefaultInputs)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		tflog.Debug(ctx, "Script execution failed", map[string]interface{}{
			"stdout":   result.MaskedStdout,
			"stderr":   result.MaskedStderr,
			"exitCode": result.ExitCode,
			"error":    err.Error(),
			"payload":  maskedPayload,
		})
		return result, fmt.Errorf("script execution failed with exit code %d: %w", result.ExitCode, err)
	}

	tflog.Debug(ctx, "Script execution completed", map[string]interface{}{
		"stdout":   result.MaskedStdout,
		"stderr":   result.MaskedStderr,
		"exitCode": result.ExitCode,
		"payload":  maskedPayload,
	})

	if stdout.Len() == 0 {
		tflog.Debug(ctx, "Script output is empty")
		return result, nil
	}

	// Re-read stdout from the original string since the buffer was consumed
	var jsonResult map[string]interface{}
	d := json.NewDecoder(strings.NewReader(result.Stdout))
	if config.HighPrecisionNumbers {
		d.UseNumber()
	}
	if err := d.Decode(&jsonResult); err != nil {
		return result, fmt.Errorf("failed to parse script output: %w", err)
	}

	result.Result = jsonResult
	return result, nil
}

// WithSemaphore runs the given function with semaphore acquire/release if the semaphore is not nil.
func WithSemaphore(sem chan struct{}, fn func()) {
	if sem != nil {
		sem <- struct{}{}
		defer func() { <-sem }()
	}
	fn()
}

// MaskJSONString parses a JSON string and replaces values of sensitive keys with "***",
// then re-serializes it. If parsing fails, falls back to string-based replacement.
func MaskJSONString(jsonStr string, sensitiveKeys []string) string {
	if len(sensitiveKeys) == 0 {
		return jsonStr
	}
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return jsonStr
	}
	keySet := make(map[string]bool, len(sensitiveKeys))
	for _, k := range sensitiveKeys {
		keySet[k] = true
	}
	maskRecursive(data, keySet)
	masked, err := json.Marshal(data)
	if err != nil {
		return jsonStr
	}
	return string(masked)
}

func maskRecursive(data interface{}, sensitiveKeys map[string]bool) {
	switch v := data.(type) {
	case map[string]interface{}:
		for key := range v {
			if sensitiveKeys[key] {
				v[key] = "***"
			} else {
				maskRecursive(v[key], sensitiveKeys)
			}
		}
	case []interface{}:
		for _, elem := range v {
			maskRecursive(elem, sensitiveKeys)
		}
	}
}

// MaskSensitiveValues replaces occurrences of sensitive values in a string with "***".
// This is used for stderr where the output format is not guaranteed to be JSON.
func MaskSensitiveValues(s string, sensitiveInputs interface{}) string {
	if sensitiveInputs == nil {
		return s
	}
	values := collectStringValues(sensitiveInputs)
	for _, val := range values {
		if val != "" {
			s = strings.ReplaceAll(s, val, "***")
		}
	}
	return s
}

func collectStringValues(data interface{}) []string {
	var values []string
	switch v := data.(type) {
	case map[string]interface{}:
		for _, val := range v {
			values = append(values, collectStringValues(val)...)
		}
	case []interface{}:
		for _, elem := range v {
			values = append(values, collectStringValues(elem)...)
		}
	case string:
		values = append(values, v)
	}
	return values
}
