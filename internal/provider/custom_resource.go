// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &CustomResource{}
var _ resource.ResourceWithImportState = &CustomResource{}

func NewCustomResource() resource.Resource {
	return &CustomResource{}
}

// ExampleResource defines the resource implementation.
type CustomResource struct{}

// ExampleResourceModel describes the resource data model.
type CustomResourceModel struct {
	CreateScript types.List `tfsdk:"create_script"`
	ReadScript   types.List `tfsdk:"read_script"`
	UpdateScript types.List `tfsdk:"update_script"`
	DeleteScript types.List `tfsdk:"delete_script"`
	Input        types.Map  `tfsdk:"input"`
	Output       types.Map  `tfsdk:"output"`
}

func (r *CustomResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *CustomResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Custom resource for custom CRUD operations",

		Attributes: map[string]schema.Attribute{
			"create_script": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Script to run for create operation",
			},
			"read_script": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Script to run for read operation",
			},
			"update_script": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				MarkdownDescription: "Script to run for update operation",
			},
			"delete_script": schema.ListAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Script to run for delete operation",
			},
			"input": schema.MapAttribute{
				ElementType:         types.StringType,
				Required:            true,
				MarkdownDescription: "Input map to pass to the scripts",
			},
			"output": schema.MapAttribute{
				ElementType:         types.StringType,
				Computed:            true,
				MarkdownDescription: "Output values from the scripts",
			},
		},
	}
}

func (r *CustomResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomResourceModel

	// Get plan values
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize empty map if input is null
	if data.Input.IsNull() {
		mapVal, diags := types.MapValue(types.StringType, map[string]attr.Value{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Input = mapVal
	}

	// Convert input map to JSON string for script execution
	var inputMap map[string]string
	data.Input.ElementsAs(ctx, &inputMap, false)
	inputJSON, err := json.Marshal(inputMap)
	if err != nil {
		resp.Diagnostics.AddError(
			"Input Conversion Failed",
			fmt.Sprintf("Failed to convert input to JSON: %s", err.Error()),
		)
		return
	}

	// Execute create script
	createOutput, err := r.executeScript(ctx, data.CreateScript, string(inputJSON))
	if err != nil {
		resp.Diagnostics.AddError(
			"Create Script Execution Failed",
			fmt.Sprintf("Failed to execute create script: %s", err.Error()),
		)
		return
	}

	tflog.Debug(ctx, "Create script output",
		map[string]interface{}{
			"output": createOutput,
		})

	// Parse output to extract ID
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(createOutput), &result); err != nil {
		resp.Diagnostics.AddError(
			"Create Script Output Parse Error",
			fmt.Sprintf("Failed to parse JSON output: %s", err.Error()),
		)
		return
	}

	// Convert the remaining state to a map for storage
	resultMap := make(map[string]attr.Value)
	for k, v := range result {
		if str, ok := v.(string); ok {
			resultMap[k] = types.StringValue(str)
		} else {
			strVal, _ := json.Marshal(v)
			resultMap[k] = types.StringValue(string(strVal))
		}
	}

	// Update input map with matching values from result
	inputResultMap := make(map[string]attr.Value)
	var currentInput map[string]string
	data.Input.ElementsAs(ctx, &currentInput, false)
	for k := range currentInput {
		if v, ok := resultMap[k]; ok {
			inputResultMap[k] = v
		} else {
			inputResultMap[k] = types.StringValue(currentInput[k])
		}
	}

	// Create input and output maps
	inputMapVal, diags := types.MapValue(types.StringType, inputResultMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Input = inputMapVal

	outputMapVal, diags := types.MapValue(types.StringType, resultMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Output = outputMapVal

	tflog.Debug(ctx, "Setting state with create output",
		map[string]interface{}{
			"input":  inputResultMap,
			"output": resultMap,
		})

	// Save the data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomResourceModel

	// Get current state
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize empty map if input is null
	if data.Input.IsNull() {
		mapVal, diags := types.MapValue(types.StringType, map[string]attr.Value{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Input = mapVal
	}

	// Convert input and output maps to string maps for script execution
	var inputMap, outputMap map[string]string
	data.Input.ElementsAs(ctx, &inputMap, false)
	data.Output.ElementsAs(ctx, &outputMap, false)

	// Create a separate map for script execution that includes both input and output
	scriptInput := make(map[string]string)
	for k, v := range inputMap {
		scriptInput[k] = v
	}
	for k, v := range outputMap {
		scriptInput[k] = v
	}

	readInputJSON, err := json.Marshal(scriptInput)
	if err != nil {
		resp.Diagnostics.AddError(
			"Input Conversion Failed",
			fmt.Sprintf("Failed to convert input to JSON: %s", err.Error()),
		)
		return
	}

	// Execute read script
	readOutput, err := r.executeScript(ctx, data.ReadScript, string(readInputJSON))
	if err != nil {
		resp.Diagnostics.AddError(
			"Read Script Execution Failed",
			fmt.Sprintf("Failed to execute read script: %s", err.Error()),
		)
		return
	}

	// Parse the read output
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(readOutput), &result); err != nil {
		resp.Diagnostics.AddError(
			"Read Script Output Parse Error",
			fmt.Sprintf("Failed to parse JSON output: %s", err.Error()),
		)
		return
	}

	// Convert to map values and store everything from the script output except ID
	resultMap := make(map[string]attr.Value)
	for k, v := range result {
		if str, ok := v.(string); ok {
			resultMap[k] = types.StringValue(str)
		} else {
			strVal, _ := json.Marshal(v)
			resultMap[k] = types.StringValue(string(strVal))
		}
	}

	// Update input map with matching values from result
	inputResultMap := make(map[string]attr.Value)
	var currentInput map[string]string
	data.Input.ElementsAs(ctx, &currentInput, false)
	for k := range currentInput {
		if v, ok := resultMap[k]; ok {
			inputResultMap[k] = v
		} else {
			inputResultMap[k] = types.StringValue(currentInput[k])
		}
	}

	// Create input and output maps
	inputMapVal, diags := types.MapValue(types.StringType, inputResultMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Input = inputMapVal

	outputMapVal, diags := types.MapValue(types.StringType, resultMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.Output = outputMapVal

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CustomResourceModel
	var state CustomResourceModel

	// Read Terraform state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Initialize empty map if input is null
	if data.Input.IsNull() {
		mapVal, diags := types.MapValue(types.StringType, map[string]attr.Value{})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Input = mapVal
	}

	scriptInputMap := make(map[string]string)

	outputMap := state.Output.Elements()
	for key, val := range outputMap {
		strVal, ok := val.(types.String)
		if ok && !strVal.IsUnknown() && !strVal.IsNull() {
			scriptInputMap[key] = strVal.ValueString()
			tflog.Debug(ctx, "Update input map from output", map[string]interface{}{
				"key":   key,
				"value": strVal.ValueString(),
			})
		}
	}

	inputMap := data.Input.Elements()
	for key, val := range inputMap {
		strVal, ok := val.(types.String)
		if ok && !strVal.IsUnknown() && !strVal.IsNull() {
			scriptInputMap[key] = strVal.ValueString()
			tflog.Debug(ctx, "Update input map from input", map[string]interface{}{
				"key":   key,
				"value": strVal.ValueString(),
			})
		}
	}

	updateInputJSON, err := json.Marshal(scriptInputMap)
	tflog.Debug(ctx, "Update input JSON", map[string]interface{}{
		"input": scriptInputMap,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Input Conversion Failed",
			fmt.Sprintf("Failed to convert input to JSON: %s", err.Error()),
		)
		return
	}

	// Execute update script if provided
	if !data.UpdateScript.IsNull() {
		updateOutput, err := r.executeScript(ctx, data.UpdateScript, string(updateInputJSON))
		if err != nil {
			resp.Diagnostics.AddError(
				"Update Script Execution Failed",
				fmt.Sprintf("Failed to execute update script: %s", err.Error()),
			)
			return
		}

		// Parse the update output
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(updateOutput), &result); err != nil {
			resp.Diagnostics.AddError(
				"Update Script Output Parse Error",
				fmt.Sprintf("Failed to parse JSON output: %s", err.Error()),
			)
			return
		}

		// Convert to map values
		resultMap := make(map[string]attr.Value)
		for k, v := range result {
			if str, ok := v.(string); ok {
				resultMap[k] = types.StringValue(str)
			} else {
				strVal, _ := json.Marshal(v)
				resultMap[k] = types.StringValue(string(strVal))
			}
		}

		// Update input map with matching values from result
		inputResultMap := make(map[string]attr.Value)
		var currentInput map[string]string
		data.Input.ElementsAs(ctx, &currentInput, false)
		for k := range currentInput {
			if v, ok := resultMap[k]; ok {
				inputResultMap[k] = v
			} else {
				inputResultMap[k] = types.StringValue(currentInput[k])
			}
		}

		// Create input and output maps
		inputMapVal, diags := types.MapValue(types.StringType, inputResultMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Input = inputMapVal

		outputMapVal, diags := types.MapValue(types.StringType, resultMap)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.Output = outputMapVal
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
}

func (r *CustomResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert input map to JSON string for script execution
	var inputMap map[string]string
	data.Input.ElementsAs(ctx, &inputMap, false)

	deleteInputJSON, err := json.Marshal(inputMap)
	if err != nil {
		resp.Diagnostics.AddError(
			"Input Conversion Failed",
			fmt.Sprintf("Failed to convert input to JSON: %s", err.Error()),
		)
		return
	}

	// Execute delete script
	_, err = r.executeScript(ctx, data.DeleteScript, string(deleteInputJSON))
	if err != nil {
		resp.Diagnostics.AddError(
			"Delete Script Execution Failed",
			fmt.Sprintf("Failed to execute delete script: %s", err.Error()),
		)
		return
	}
}

func (r *CustomResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data CustomResourceModel

	var importObj map[string]interface{}
	err := json.Unmarshal([]byte(req.ID), &importObj)
	if err != nil {
		resp.Diagnostics.AddError(
			"Input Conversion Failed",
			fmt.Sprintf("Failed to convert input to JSON: %s", err.Error()),
		)
		return
	}

	getScriptList := func(key string) types.List {
		if val, ok := importObj[key]; ok {
			if arr, ok := val.([]interface{}); ok {
				var values []attr.Value
				for _, v := range arr {
					if s, ok := v.(string); ok {
						values = append(values, types.StringValue(s))
					}
				}
				list, diags := types.ListValue(types.StringType, values)
				if diags.HasError() {
					return types.ListNull(types.StringType)
				}
				return list
			}
		}
		return types.ListNull(types.StringType)
	}

	data.CreateScript = getScriptList("create_script")
	data.ReadScript = getScriptList("read_script")
	data.DeleteScript = getScriptList("delete_script")
	data.UpdateScript = getScriptList("update_script")

	if outputVal, ok := importObj["output"]; ok {
		if outputMap, ok := outputVal.(map[string]interface{}); ok {
			outputAttrMap := make(map[string]attr.Value)
			for k, v := range outputMap {
				if s, ok := v.(string); ok {
					outputAttrMap[k] = types.StringValue(s)
				}
			}
			outputMapVal, diags := types.MapValueFrom(ctx, types.StringType, outputAttrMap)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}
			data.Output = outputMapVal
		}
	}
	data.Input = types.MapNull(types.StringType)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// executeScript runs the provided script with input and returns the output.
func (r *CustomResource) executeScript(ctx context.Context, scriptList types.List, input string) (string, error) {
	tflog.Debug(ctx, "Executing script", map[string]interface{}{
		"script": scriptList.String(),
		"input":  input,
	})

	// Convert the script list to string array
	var scriptParts []string
	diags := scriptList.ElementsAs(ctx, &scriptParts, false)
	if diags.HasError() {
		return "", fmt.Errorf("failed to parse script command: %v", diags)
	}

	if len(scriptParts) == 0 {
		return "", fmt.Errorf("empty script command")
	}

	// Check if the first part is a file path
	if info, err := os.Stat(scriptParts[0]); err == nil {
		// Check if the file is executable
		if info.Mode()&0111 == 0 {
			return "", fmt.Errorf("script file is not executable: %s", scriptParts[0])
		}
	}

	// Execute the command with its arguments
	cmd := exec.CommandContext(ctx, scriptParts[0], scriptParts[1:]...)

	// Set up pipes for stdin, stdout, and stderr
	cmd.Stdin = strings.NewReader(input)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start command: %w", err)
	}

	// Read stdout
	stdoutBytes := make([]byte, 0)
	stdoutBuf := make([]byte, 1024)
	for {
		n, err := stdout.Read(stdoutBuf)
		if n > 0 {
			stdoutBytes = append(stdoutBytes, stdoutBuf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Read stderr
	stderrBytes := make([]byte, 0)
	stderrBuf := make([]byte, 1024)
	for {
		n, err := stderr.Read(stderrBuf)
		if n > 0 {
			stderrBytes = append(stderrBytes, stderrBuf[:n]...)
		}
		if err != nil {
			break
		}
	}

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Log stderr if there's an error
		if len(stderrBytes) > 0 {
			tflog.Error(ctx, "Script stderr output", map[string]interface{}{
				"stderr": string(stderrBytes),
			})
		}
		return "", fmt.Errorf("script execution failed: %w", err)
	}

	// Log stderr as info if command succeeded but there's stderr output
	if len(stderrBytes) > 0 {
		tflog.Info(ctx, "Script stderr output", map[string]interface{}{
			"stderr": string(stderrBytes),
		})
	}

	output := string(stdoutBytes)
	tflog.Debug(ctx, "Script executed successfully", map[string]interface{}{
		"output": output,
	})

	return output, nil
}
