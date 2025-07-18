package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type ScriptPayload struct {
	Id     string      `json:"id,omitempty"`
	Input  interface{} `json:"input,omitempty"`
	Output interface{} `json:"output,omitempty"`
}

type ScriptResult struct {
	Result   map[string]interface{}
	Stdout   string
	Stderr   string
	ExitCode int
}

// ExecuteScript runs the given command with the provided payload, returning the result and any error.
func ExecuteScript(ctx context.Context, cmd []string, payload ScriptPayload) (*ScriptResult, error) {
	if len(cmd) == 0 {
		return nil, fmt.Errorf("empty command")
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	tflog.Debug(ctx, "Executing script", map[string]interface{}{
		"command": cmd,
		"payload": string(payloadBytes),
	})

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Stdin = bytes.NewReader(payloadBytes)

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err = execCmd.Run()
	result := &ScriptResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: 0,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		tflog.Debug(ctx, "Script execution failed", map[string]interface{}{
			"stdout":   result.Stdout,
			"stderr":   result.Stderr,
			"exitCode": result.ExitCode,
			"error":    err.Error(),
			"payload":  string(payloadBytes),
		})
		return result, fmt.Errorf("script execution failed with exit code %d: %w", result.ExitCode, err)
	}

	tflog.Debug(ctx, "Script execution completed", map[string]interface{}{
		"stdout":   result.Stdout,
		"stderr":   result.Stderr,
		"exitCode": result.ExitCode,
		"payload":  string(payloadBytes),
	})

	if stdout.Len() == 0 {
		tflog.Debug(ctx, "Script output is empty")
		return result, nil
	}

	var jsonResult map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &jsonResult); err != nil {
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
