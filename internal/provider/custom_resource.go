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
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &customCrudResource{}
var _ resource.ResourceWithImportState = &customCrudResource{}

// CustomCrudResource implementation
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
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *customCrudResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:    true,
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

type scriptPayload struct {
	Id     string      `json:"id"`
	Input  interface{} `json:"input"`
	Output interface{} `json:"output"`
}

func (r *customCrudResource) convertToPayload(plan *customCrudResourceModel, state *customCrudResourceModel) ([]byte, error) {
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

	payload := scriptPayload{
		Id:     id,
		Input:  inputValue,
		Output: outputValue,
	}

	return json.Marshal(payload)
}

func (r *customCrudResource) executeScript(ctx context.Context, cmd []string, payload scriptPayload) (map[string]interface{}, error) {
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
	if err != nil {
		tflog.Debug(ctx, "Script execution failed", map[string]interface{}{
			"stdout": stdout.String(),
			"stderr": stderr.String(),
			"error":  err.Error(),
		})
		return nil, fmt.Errorf("script execution failed: %w", err)
	}

	tflog.Debug(ctx, "Script execution completed", map[string]interface{}{
		"stdout": stdout.String(),
		"stderr": stderr.String(),
	})

	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse script output: %w", err)
	}

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
	if delete, ok := attrs["delete"].(types.String); ok {
		crud.Delete = delete
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

	payloadBytes, err := r.convertToPayload(&data, nil)
	if err != nil {
		resp.Diagnostics.AddError("Payload Creation Failed", err.Error())
		return
	}

	var payload scriptPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		resp.Diagnostics.AddError("Payload Unmarshal Failed", err.Error())
		return
	}

	result, err := r.executeScript(ctx, createCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Create Script Failed", err.Error())
		return
	}

	if id, exists := result["id"]; exists {
		if idStr, ok := id.(string); ok {
			data.Id = types.StringValue(idStr)
		}
	}

	if data.Id.IsNull() || data.Id.ValueString() == "" {
		resp.Diagnostics.AddError("Create Script Error", "Create script must return an 'id' field")
		return
	}

	outputValue := r.mapToDynamic(result)
	data.Output = outputValue

	// Update input with any matching keys from output
	if !data.Input.IsNull() && !data.Input.IsUnknown() {
		updatedInput := r.mergeInputWithOutput(data.Input, result)
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

	payloadBytes, err := r.convertToPayload(nil, &data)
	if err != nil {
		resp.Diagnostics.AddError("Payload Creation Failed", err.Error())
		return
	}

	var payload scriptPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		resp.Diagnostics.AddError("Payload Unmarshal Failed", err.Error())
		return
	}

	result, err := r.executeScript(ctx, readCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Read Script Failed", err.Error())
		return
	}

	newOutput := r.mapToDynamic(result)
	data.Output = newOutput
	data.Input = r.mergeInputWithOutput(data.Input, result)

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

	if crud.Update.IsNull() {
		resp.Diagnostics.AddError("Update Not Supported", "No update command provided, resource requires recreation")
		return
	}

	updateCmd := strings.Fields(crud.Update.ValueString())
	if len(updateCmd) == 0 {
		resp.Diagnostics.AddError("Invalid Update Command", "Update command cannot be empty")
		return
	}

	payloadBytes, err := r.convertToPayload(&plan, &state)
	if err != nil {
		resp.Diagnostics.AddError("Payload Creation Failed", err.Error())
		return
	}

	var payload scriptPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		resp.Diagnostics.AddError("Payload Unmarshal Failed", err.Error())
		return
	}

	result, err := r.executeScript(ctx, updateCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Update Script Failed", err.Error())
		return
	}

	newOutput := r.mapToDynamic(result)
	plan.Output = newOutput

	if id, exists := result["id"]; exists {
		if idStr, ok := id.(string); ok {
			plan.Id = types.StringValue(idStr)
		} else {
			resp.Diagnostics.AddError("Invalid ID Type", "ID returned from update script must be a string")
			return
		}
	} else {
		plan.Id = state.Id
	}

	// Update input with any matching keys from output
	if !plan.Input.IsNull() && !plan.Input.IsUnknown() {
		updatedInput := r.mergeInputWithOutput(plan.Input, result)
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

	payloadBytes, err := r.convertToPayload(nil, &data)
	if err != nil {
		resp.Diagnostics.AddError("Payload Creation Failed", err.Error())
		return
	}

	var payload scriptPayload
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		resp.Diagnostics.AddError("Payload Unmarshal Failed", err.Error())
		return
	}

	_, err = r.executeScript(ctx, deleteCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Delete Script Failed", err.Error())
		return
	}
}

func (r *customCrudResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts := strings.Split(req.ID, ":")
	if len(parts) < 2 {
		resp.Diagnostics.AddError("Invalid Import ID", "Import ID must be in format 'create_cmd,read_cmd,update_cmd,delete_cmd:id'")
		return
	}

	id := parts[len(parts)-1]
	cmds := strings.Split(parts[0], ",")

	if len(cmds) < 3 {
		resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Must provide at least create, read, and delete commands, received: %+v", cmds))
		return
	}

	// Create the hooks block object
	hooksAttrs := map[string]attr.Value{
		"create": types.StringValue(cmds[0]),
		"read":   types.StringValue(cmds[1]),
		"update": types.StringValue(cmds[2]),
		"delete": types.StringValue(cmds[3]),
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
		Id:    types.StringValue(id),
		Hooks: hooksList,
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
		Id:     id,
		Input:  nil,
		Output: nil,
	}

	// Use read to populate the state
	result, err := r.executeScript(ctx, readCmd, payload)
	if err != nil {
		resp.Diagnostics.AddError("Import Read Failed", err.Error())
		return
	}

	outputValue := r.mapToDynamic(result)
	data.Output = outputValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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
	case bool:
		return types.BoolValue(v)
	case []interface{}:
		elements := make([]attr.Value, len(v))
		var elemType attr.Type = types.StringType // default
		for i, elem := range v {
			elements[i] = r.interfaceToAttrValue(elem)
			if i == 0 {
				elemType = elements[i].Type(context.Background())
			}
		}
		listVal, _ := types.ListValue(elemType, elements)
		return listVal
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
		return types.StringNull()
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
	default:
		return nil
	}
}
