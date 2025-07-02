// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os/exec"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &customCrudResource{}
var _ resource.ResourceWithImportState = &customCrudResource{}
var _ resource.ResourceWithModifyPlan = &customCrudResource{}

// CustomCrudResource implementation.
type customCrudResourceModel struct {
	Id     types.String  `tfsdk:"id"`
	Hooks  types.List    `tfsdk:"hooks"`
	Input  types.Dynamic `tfsdk:"input"`
	Output types.Dynamic `tfsdk:"output"`
}

type hooksBlockValue struct {
	Create types.String `tfsdk:"create"`
	Read   types.String `tfsdk:"read"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

type customCrudResource struct{}

func NewCustomCrudResource() resource.Resource {
	return &customCrudResource{}
}

func (r *customCrudResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = "customcrud"
}

func (r *customCrudResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "Resource identifier",
			},
			"input": schema.DynamicAttribute{
				Optional:    true,
				Description: "Input data for the resource",
			},
			"output": schema.DynamicAttribute{
				Computed:    true,
				Description: "Output data from the resource",
			},
		},
		Blocks: map[string]schema.Block{
			"hooks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"create": schema.StringAttribute{
							Required:    true,
							Description: "Create command (space-separated command and arguments)",
						},
						"read": schema.StringAttribute{
							Required:    true,
							Description: "Read command (space-separated command and arguments)",
						},
						"update": schema.StringAttribute{
							Optional:    true,
							Description: "Update command (space-separated command and arguments)",
						},
						"delete": schema.StringAttribute{
							Required:    true,
							Description: "Delete command (space-separated command and arguments)",
						},
					},
				},
				Validators: []validator.List{
					listvalidator.SizeAtMost(1),
				},
			},
		},
	}
}

// ModifyPlan implements resource.ResourceWithModifyPlan to force replacement
// when update hook is not provided and input has changed.
func (r *customCrudResource) ModifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	// Only process during updates (not create or delete)
	if req.State.Raw.IsNull() || req.Plan.Raw.IsNull() {
		return
	}

	var state, plan customCrudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get CRUD commands from the plan
	crud, err := r.getCrudCommands(&plan)
	if err != nil {
		// If we can't get CRUD commands, let the normal validation handle it
		return
	}

	// If update hook is not provided (null or empty), force replacement on any input change
	if crud.Update.IsNull() || strings.TrimSpace(crud.Update.ValueString()) == "" {
		// Check if input has changed
		if !state.Input.Equal(plan.Input) {
			tflog.Debug(ctx, "Update hook not provided and input changed, forcing replacement")
			resp.RequiresReplace = append(resp.RequiresReplace, path.Root("input"))
		}
	}
}

type scriptPayload struct {
	Id     string      `json:"id"`
	Input  interface{} `json:"input"`
	Output interface{} `json:"output"`
}

type scriptResult struct {
	Result   map[string]interface{}
	Stdout   string
	Stderr   string
	ExitCode int
}

func (r *customCrudResource) convertToPayload(plan *customCrudResourceModel, state *customCrudResourceModel) scriptPayload {
	var inputValue interface{}
	var outputValue interface{}
	id := ""

	// Get input from plan or state
	if plan != nil && !plan.Input.IsNull() && !plan.Input.IsUnknown() {
		inputValue = r.attrValueToInterface(plan.Input.UnderlyingValue())
	} else if state != nil && !state.Input.IsNull() && !state.Input.IsUnknown() {
		inputValue = r.attrValueToInterface(state.Input.UnderlyingValue())
	}

	// Get output and id from state
	if state != nil {
		id = state.Id.ValueString()
		if !state.Output.IsNull() && !state.Output.IsUnknown() {
			outputValue = r.attrValueToInterface(state.Output.UnderlyingValue())
		}
	}

	return scriptPayload{
		Id:     id,
		Input:  inputValue,
		Output: outputValue,
	}
}

func (r *customCrudResource) executeScript(ctx context.Context, cmd []string, payload scriptPayload) (*scriptResult, error) {
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
	result := &scriptResult{
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

func (r *customCrudResource) getCrudCommands(data *customCrudResourceModel) (*hooksBlockValue, error) {
	if data.Hooks.IsNull() || data.Hooks.IsUnknown() {
		return nil, fmt.Errorf("crud block is null or unknown")
	}

	elements := data.Hooks.Elements()
	if len(elements) == 0 {
		return nil, fmt.Errorf("crud block is empty")
	}

	obj, ok := elements[0].(types.Object)
	if !ok {
		return nil, fmt.Errorf("crud block element is not an object")
	}

	crud := &hooksBlockValue{}
	attrs := obj.Attributes()

	if create, ok := attrs["create"].(types.String); ok {
		crud.Create = create
	}
	if read, ok := attrs["read"].(types.String); ok {
		crud.Read = read
	}
	if update, ok := attrs["update"].(types.String); ok {
		crud.Update = update
	}
	if destroy, ok := attrs["delete"].(types.String); ok {
		crud.Delete = destroy // delete is a reserved keyword in Go, so we use "destroy" here
	}

	return crud, nil
}

func (r *customCrudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data customCrudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	crud, err := r.getCrudCommands(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting CRUD commands", err.Error())
		return
	}

	createCmd := strings.Fields(crud.Create.ValueString())
	if len(createCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Create Command", "Create command cannot be empty")
		return
	}

	payload := r.convertToPayload(&data, nil)
	result, err := r.executeScript(ctx, createCmd, payload)
	if err != nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Create Script Failed", fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", err, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}
	if result == nil || result.Result == nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Create Script Failed", fmt.Sprintf("Create script returned nil output\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}

	if id, exists := result.Result["id"]; exists {
		if idStr, ok := id.(string); ok {
			data.Id = types.StringValue(idStr)
		} else {
			// convert to string if necessary
			idStr = fmt.Sprintf("%v", id)
			data.Id = types.StringValue(idStr)
		}
	}

	if data.Id.IsNull() || data.Id.ValueString() == "" {
		resp.Diagnostics.AddError("Create Script Error", "Create script must return an 'id' field")
		return
	}

	outputValue := r.mapToDynamic(result.Result)
	data.Output = outputValue

	// Update input with any matching keys from output
	if !data.Input.IsNull() && !data.Input.IsUnknown() {
		updatedInput := r.mergeInputWithOutput(data.Input, result.Result)
		data.Input = updatedInput
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customCrudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data customCrudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	crud, err := r.getCrudCommands(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting CRUD commands", err.Error())
		return
	}

	readCmd := strings.Fields(crud.Read.ValueString())
	if len(readCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Read Command", "Read command cannot be empty")
		return
	}

	payload := r.convertToPayload(nil, &data)
	result, err := r.executeScript(ctx, readCmd, payload)
	if err != nil {
		payloadJSON, _ := json.Marshal(payload)
		// Treat exit code 22 from script as a signal to recreate resource
		// and return early
		if result.ExitCode == 22 {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Read Script Failed", fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", err, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}
	if result == nil || result.Result == nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Read Script Failed", fmt.Sprintf("Read script returned nil output\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}

	newOutput := r.mapToDynamic(result.Result)
	data.Output = newOutput
	data.Input = r.mergeInputWithOutput(data.Input, result.Result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customCrudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan customCrudResourceModel
	var state customCrudResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	crud, err := r.getCrudCommands(&plan)
	if err != nil {
		resp.Diagnostics.AddError("Error getting CRUD commands", err.Error())
		return
	}

	updateCmd := strings.Fields(crud.Update.ValueString())
	if len(updateCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Update Command", "Update command cannot be empty")
		return
	}

	payload := r.convertToPayload(&plan, &state)
	result, err := r.executeScript(ctx, updateCmd, payload)
	if err != nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Update Script Failed", fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", err, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}
	if result == nil || result.Result == nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Update Script Failed", fmt.Sprintf("Update script returned nil output\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}

	newOutput := r.mapToDynamic(result.Result)
	plan.Output = newOutput

	if id, exists := result.Result["id"]; exists {
		if idStr, ok := id.(string); ok {
			plan.Id = types.StringValue(idStr)
		} else {
			// convert to string if necessary
			idStr = fmt.Sprintf("%v", id)
			plan.Id = types.StringValue(idStr)
		}
	} else {
		plan.Id = state.Id
	}

	// Update input with any matching keys from output
	if !plan.Input.IsNull() && !plan.Input.IsUnknown() {
		updatedInput := r.mergeInputWithOutput(plan.Input, result.Result)
		plan.Input = updatedInput
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *customCrudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data customCrudResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	crud, err := r.getCrudCommands(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting CRUD commands", err.Error())
		return
	}

	deleteCmd := strings.Fields(crud.Delete.ValueString())
	if len(deleteCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Delete Command", "Delete command cannot be empty")
		return
	}

	payload := r.convertToPayload(nil, &data)
	result, err := r.executeScript(ctx, deleteCmd, payload)
	if err != nil {
		payloadJSON, _ := json.Marshal(payload)
		resp.Diagnostics.AddError("Delete Script Failed", fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", err, result.ExitCode, result.Stdout, result.Stderr, string(payloadJSON)))
		return
	}
}

type importStateData struct {
	Id     string                 `json:"id"`
	Hooks  map[string]string      `json:"hooks"`
	Input  map[string]interface{} `json:"input"`
	Output map[string]interface{} `json:"output"`
}

func (r *customCrudResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var importData importStateData
	if err := json.Unmarshal([]byte(req.ID), &importData); err != nil {
		resp.Diagnostics.AddError("Invalid Import JSON", fmt.Sprintf("Failed to parse import JSON: %v. Import ID must be a JSON string containing id, hooks, input, and output fields.", err))
		return
	}

	if importData.Id == "" {
		resp.Diagnostics.AddError("Invalid Import JSON", "Import JSON must contain a non-empty 'id' field")
		return
	}

	if len(importData.Hooks) < 3 {
		resp.Diagnostics.AddError("Invalid Import JSON", "Import JSON must contain hooks with at least create, read, and delete commands")
		return
	}

	hooksAttrs := map[string]attr.Value{
		"create": types.StringValue(importData.Hooks["create"]),
		"read":   types.StringValue(importData.Hooks["read"]),
		"delete": types.StringValue(importData.Hooks["delete"]),
	}

	// Add update command if provided
	if updateCmd, ok := importData.Hooks["update"]; ok {
		hooksAttrs["update"] = types.StringValue(updateCmd)
	} else {
		hooksAttrs["update"] = types.StringNull()
	}

	hooksObj, diags := types.ObjectValue(
		map[string]attr.Type{
			"create": types.StringType,
			"read":   types.StringType,
			"update": types.StringType,
			"delete": types.StringType,
		},
		hooksAttrs,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hooksList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"create": types.StringType,
				"read":   types.StringType,
				"update": types.StringType,
				"delete": types.StringType,
			},
		},
		[]attr.Value{hooksObj},
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data := customCrudResourceModel{
		Id:    types.StringValue(importData.Id),
		Hooks: hooksList,
	}

	if importData.Input != nil {
		data.Input = r.mapToDynamic(importData.Input)
	}

	if importData.Output != nil {
		data.Output = r.mapToDynamic(importData.Output)
	}

	crud, err := r.getCrudCommands(&data)
	if err != nil {
		resp.Diagnostics.AddError("Error getting CRUD commands", err.Error())
		return
	}

	readCmd := strings.Fields(crud.Read.ValueString())
	if len(readCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Read Command", "Read command cannot be empty")
		return
	}

	payload := scriptPayload{
		Id:     importData.Id,
		Input:  importData.Input,
		Output: importData.Output,
	}

	// Use read to populate the state
	result, err := r.executeScript(ctx, readCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Import Read Failed", fmt.Sprintf("%v\nExit Code: %d\nStdout: %s\nStderr: %s", err, result.ExitCode, result.Stdout, result.Stderr))
		return
	}
	if result == nil || result.Result == nil {
		resp.Diagnostics.AddError("Import Read Failed", fmt.Sprintf("Import read script returned nil output\nExit Code: %d\nStdout: %s\nStderr: %s", result.ExitCode, result.Stdout, result.Stderr))
		return
	}

	outputValue := r.mapToDynamic(result.Result)
	data.Output = outputValue
	data.Input = r.mergeInputWithOutput(data.Input, result.Result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customCrudResource) isSimpleType(t attr.Type) bool {
	switch t.(type) {
	case basetypes.StringType, basetypes.NumberType, basetypes.BoolType:
		return true
	default:
		return false
	}
}

func (r *customCrudResource) mapToDynamic(data interface{}) types.Dynamic {
	switch v := data.(type) {
	case map[string]interface{}:
		attrs := make(map[string]attr.Value)
		for k, val := range v {
			attrs[k] = r.interfaceToAttrValue(val)
		}
		objType := types.ObjectType{AttrTypes: make(map[string]attr.Type)}
		for k := range attrs {
			objType.AttrTypes[k] = attrs[k].Type(context.Background())
		}
		objVal, _ := types.ObjectValue(objType.AttrTypes, attrs)
		return types.DynamicValue(objVal)
	default:
		return types.DynamicValue(r.interfaceToAttrValue(v))
	}
}

func (r *customCrudResource) interfaceToAttrValue(data interface{}) attr.Value {
	switch v := data.(type) {
	case string:
		return types.StringValue(v)
	case float64:
		return types.NumberValue(big.NewFloat(v))
	case int:
		return types.NumberValue(big.NewFloat(float64(v)))
	case bool:
		return types.BoolValue(v)
	case []interface{}:
		if len(v) == 0 {
			// Empty array - default to dynamic
			listVal, _ := types.ListValue(types.DynamicType, []attr.Value{})
			return listVal
		}

		elements := make([]attr.Value, len(v))

		// First pass: convert all elements
		for i, elem := range v {
			elements[i] = r.interfaceToAttrValue(elem)
		}

		// Check if all elements have the same type
		firstType := elements[0].Type(context.Background())
		isHomogeneous := true
		allSimpleTypes := r.isSimpleType(firstType)

		// Check if all elements have the same type
		for i := 1; i < len(elements); i++ {
			currentType := elements[i].Type(context.Background())
			if !currentType.Equal(firstType) {
				isHomogeneous = false
			}
			// Also verify all elements are simple types
			if !r.isSimpleType(currentType) {
				allSimpleTypes = false
			}
		}

		if isHomogeneous && allSimpleTypes {
			// Homogeneous simple types - use typed list
			listVal, _ := types.ListValue(firstType, elements)
			return listVal
		} else {
			// Heterogeneous types - use tuple
			elemTypes := make([]attr.Type, len(elements))
			for i, elem := range elements {
				elemTypes[i] = elem.Type(context.Background())
			}
			tupleVal, _ := types.TupleValue(elemTypes, elements)
			return tupleVal
		}
	case map[string]interface{}:
		attrs := make(map[string]attr.Value)
		attrTypes := make(map[string]attr.Type)
		for k, val := range v {
			attrs[k] = r.interfaceToAttrValue(val)
			attrTypes[k] = attrs[k].Type(context.Background())
		}
		objVal, _ := types.ObjectValue(attrTypes, attrs)
		return objVal
	case nil:
		return types.DynamicNull()
	default:
		return types.StringValue(fmt.Sprintf("%v", v))
	}
}

func (r *customCrudResource) mergeInputWithOutput(input types.Dynamic, output map[string]interface{}) types.Dynamic {
	if input.IsNull() || input.IsUnknown() {
		return input
	}

	// Convert input to map[string]interface{} via JSON marshaling/unmarshaling
	inputMap := r.attrValueToInterface(input.UnderlyingValue())
	inputMapTyped, ok := inputMap.(map[string]interface{})
	if !ok {
		return input
	}

	merged := make(map[string]interface{})
	for k, v := range inputMapTyped {
		merged[k] = v
	}

	// Update input values with matching output keys
	for k, v := range output {
		if _, exists := merged[k]; exists {
			merged[k] = v
		}
	}

	return r.mapToDynamic(merged)
}

func (r *customCrudResource) attrValueToInterface(val attr.Value) interface{} {
	switch v := val.(type) {
	case types.String:
		if v.IsNull() {
			return nil
		}
		return v.ValueString()
	case types.Number:
		if v.IsNull() {
			return nil
		}
		f, _ := v.ValueBigFloat().Float64()
		return f
	case types.Bool:
		if v.IsNull() {
			return nil
		}
		return v.ValueBool()
	case types.List:
		if v.IsNull() {
			return nil
		}
		elements := v.Elements()
		result := make([]interface{}, len(elements))
		for i, elem := range elements {
			// Handle dynamic elements in lists
			if dynamicElem, ok := elem.(types.Dynamic); ok {
				result[i] = r.attrValueToInterface(dynamicElem.UnderlyingValue())
			} else {
				result[i] = r.attrValueToInterface(elem)
			}
		}
		return result
	case types.Tuple:
		if v.IsNull() {
			return nil
		}
		elements := v.Elements()
		result := make([]interface{}, len(elements))
		for i, elem := range elements {
			result[i] = r.attrValueToInterface(elem)
		}
		return result
	case types.Object:
		if v.IsNull() {
			return nil
		}
		attrs := v.Attributes()
		result := make(map[string]interface{})
		for k, attr := range attrs {
			result[k] = r.attrValueToInterface(attr)
		}
		return result
	case types.Dynamic:
		if v.IsNull() || v.IsUnknown() {
			return nil
		}
		return r.attrValueToInterface(v.UnderlyingValue())
	default:
		return nil
	}
}
