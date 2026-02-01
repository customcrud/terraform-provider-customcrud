// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &customCrudResource{}
var _ resource.ResourceWithImportState = &customCrudResource{}
var _ resource.ResourceWithModifyPlan = &customCrudResource{}
var _ resource.ResourceWithConfigure = &customCrudResource{}

// CustomCrudResource implementation.
type customCrudResourceModel struct {
	Id      types.String  `tfsdk:"id"`
	Hooks   types.List    `tfsdk:"hooks"`
	Input   types.Dynamic `tfsdk:"input"`
	InputWO types.String  `tfsdk:"input_wo"`
	Output  types.Dynamic `tfsdk:"output"`
}

func (m *customCrudResourceModel) GetHooks() types.List {
	return m.Hooks
}

type hooksBlockValue struct {
	Create types.String `tfsdk:"create"`
	Read   types.String `tfsdk:"read"`
	Update types.String `tfsdk:"update"`
	Delete types.String `tfsdk:"delete"`
}

type customCrudResource struct {
	config utils.CustomCRUDProviderConfig
}

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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"input": schema.DynamicAttribute{
				Optional:    true,
				Description: "Input data for the resource",
			},
			"input_wo": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				WriteOnly:   true,
				Description: "Write-only input data (JSON string) for the resource, merged with input",
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
						utils.Create: schema.StringAttribute{
							Required:    true,
							Description: "Create command (space-separated command and arguments)",
						},
						utils.Read: schema.StringAttribute{
							Required:    true,
							Description: "Read command (space-separated command and arguments)",
						},
						utils.Update: schema.StringAttribute{
							Optional:    true,
							Description: "Update command (space-separated command and arguments)",
						},
						utils.Delete: schema.StringAttribute{
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
	crud, err := getCrudCommands(&plan)
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

func getCrudCommands(data *customCrudResourceModel) (*hooksBlockValue, error) {
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

	if create, ok := attrs[utils.Create].(types.String); ok {
		crud.Create = create
	}
	if read, ok := attrs[utils.Read].(types.String); ok {
		crud.Read = read
	}
	if update, ok := attrs[utils.Update].(types.String); ok {
		crud.Update = update
	}
	if destroy, ok := attrs[utils.Delete].(types.String); ok {
		crud.Delete = destroy // delete is a reserved keyword in Go, so we use "destroy" here
	}

	return crud, nil
}

func (r *customCrudResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		r.config = utils.CustomCRUDProviderConfigDefaults()
		return
	}
	if data, ok := req.ProviderData.(*CustomCRUDProvider); ok {
		r.config = data.config
	}
}

// Helper to extract model from request and append diagnostics.
func extractModel[T any](ctx context.Context, getFn func(context.Context, any) diag.Diagnostics, diagnostics *diag.Diagnostics) (*T, bool) {
	var model T
	*diagnostics = append(*diagnostics, getFn(ctx, &model)...)
	if diagnostics.HasError() {
		return nil, false
	}
	return &model, true
}

func (r *customCrudResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	utils.WithSemaphore(r.config.Semaphore, func() {
		plan, ok := extractModel[customCrudResourceModel](ctx, req.Plan.Get, &resp.Diagnostics)
		if !ok {
			return
		}

		var config customCrudResourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if resp.Diagnostics.HasError() {
			return
		}

		payload := utils.ExecutionPayload{
			Id:     plan.Id.ValueString(),
			Input:  r.mergeInputWithWO(plan.Input, config.InputWO),
			Output: utils.AttrValueToInterface(plan.Output.UnderlyingValue()),
		}
		result, ok := utils.RunCrudScript(ctx, r.config, plan, payload, &resp.Diagnostics, utils.CrudCreate)
		if !ok {
			return
		}
		if id, exists := result.Result["id"]; exists {
			if idStr, ok := id.(string); ok {
				plan.Id = types.StringValue(idStr)
			} else {
				idStr = fmt.Sprintf("%v", id)
				plan.Id = types.StringValue(idStr)
			}
		}
		if plan.Id.IsNull() || plan.Id.ValueString() == "" {
			resp.Diagnostics.AddError(
				"Create Execution Error",
				fmt.Sprintf("Create script must return an 'id' field\nExit Code: %d\nStdout: %s\nStderr: %s\nInput Payload: %s", result.ExitCode, result.Stdout, result.Stderr, result.Payload),
			)
			return
		}
		plan.Output = utils.MapToDynamic(result.Result)
		plan.Input = r.mergeInputWithOutput(plan.Input, result.Result)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	})
}

func (r *customCrudResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	utils.WithSemaphore(r.config.Semaphore, func() {
		state, ok := extractModel[customCrudResourceModel](ctx, req.State.Get, &resp.Diagnostics)
		if !ok {
			return
		}
		payload := utils.ExecutionPayload{
			Id:     state.Id.ValueString(),
			Input:  utils.AttrValueToInterface(state.Input.UnderlyingValue()),
			Output: utils.AttrValueToInterface(state.Output.UnderlyingValue()),
		}
		result, ok := utils.RunCrudScript(ctx, r.config, state, payload, &resp.Diagnostics, utils.CrudRead)
		if !ok {
			// Special case: treat exit code 22 as resource removed
			if result != nil && result.ExitCode == 22 {
				resp.State.RemoveResource(ctx)
			}
			return
		}
		state.Output = utils.MapToDynamic(result.Result)
		state.Input = r.mergeInputWithOutput(state.Input, result.Result)
		resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
	})
}

func (r *customCrudResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	utils.WithSemaphore(r.config.Semaphore, func() {
		plan, ok := extractModel[customCrudResourceModel](ctx, req.Plan.Get, &resp.Diagnostics)
		if !ok {
			return
		}
		state, ok := extractModel[customCrudResourceModel](ctx, req.State.Get, &resp.Diagnostics)
		if !ok {
			return
		}

		var config customCrudResourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
		if resp.Diagnostics.HasError() {
			return
		}

		payload := utils.ExecutionPayload{
			Id:     plan.Id.ValueString(),
			Input:  r.mergeInputWithWO(plan.Input, config.InputWO),
			Output: utils.AttrValueToInterface(state.Output.UnderlyingValue()),
		}
		// Only run crud script if input has changed, hook changes shouldn't trigger execution
		if state.Input.Equal(plan.Input) {
			tflog.Info(ctx, "Hook-only change, skipping update execution")
			plan.Input = state.Input
			plan.Output = state.Output
			resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
			return
		}
		result, ok := utils.RunCrudScript(ctx, r.config, plan, payload, &resp.Diagnostics, utils.CrudUpdate)
		if !ok {
			return
		}
		if id, exists := result.Result["id"]; exists {
			if idStr, ok := id.(string); ok {
				plan.Id = types.StringValue(idStr)
			} else {
				idStr = fmt.Sprintf("%v", id)
				plan.Id = types.StringValue(idStr)
			}
		} else {
			plan.Id = state.Id
		}
		plan.Output = utils.MapToDynamic(result.Result)
		plan.Input = r.mergeInputWithOutput(plan.Input, result.Result)
		resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	})
}

func (r *customCrudResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	utils.WithSemaphore(r.config.Semaphore, func() {
		data, ok := extractModel[customCrudResourceModel](ctx, req.State.Get, &resp.Diagnostics)
		if !ok {
			return
		}
		payload := utils.ExecutionPayload{
			Id:     data.Id.ValueString(),
			Input:  utils.AttrValueToInterface(data.Input.UnderlyingValue()),
			Output: utils.AttrValueToInterface(data.Output.UnderlyingValue()),
		}
		_, _ = utils.RunCrudScript(ctx, r.config, data, payload, &resp.Diagnostics, utils.CrudDelete)
	})
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
		utils.Create: types.StringValue(importData.Hooks[utils.Create]),
		utils.Read:   types.StringValue(importData.Hooks[utils.Read]),
		utils.Delete: types.StringValue(importData.Hooks[utils.Delete]),
	}

	// Add update command if provided
	if updateCmd, ok := importData.Hooks[utils.Update]; ok {
		hooksAttrs[utils.Update] = types.StringValue(updateCmd)
	} else {
		hooksAttrs[utils.Update] = types.StringNull()
	}

	hooksType := map[string]attr.Type{
		utils.Create: types.StringType,
		utils.Read:   types.StringType,
		utils.Update: types.StringType,
		utils.Delete: types.StringType,
	}
	hooksObj, diags := types.ObjectValue(
		hooksType,
		hooksAttrs,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	hooksList, diags := types.ListValue(
		types.ObjectType{
			AttrTypes: hooksType,
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
		data.Input = utils.MapToDynamic(importData.Input)
	}

	if importData.Output != nil {
		data.Output = utils.MapToDynamic(importData.Output)
	}

	payload := utils.ExecutionPayload{
		Id:     importData.Id,
		Input:  importData.Input,
		Output: importData.Output,
	}

	// Use read to populate the state
	result, ok := utils.RunCrudScript(ctx, r.config, &data, payload, &resp.Diagnostics, utils.CrudRead)
	if !ok {
		return
	}

	if result == nil || result.Result == nil {
		resp.Diagnostics.AddError("Import Read Failed", "Import read script returned nil output")
		return
	}

	outputValue := utils.MapToDynamic(result.Result)
	data.Output = outputValue
	data.Input = r.mergeInputWithOutput(data.Input, result.Result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *customCrudResource) mergeInputWithOutput(input types.Dynamic, output map[string]interface{}) types.Dynamic {
	if input.IsNull() || input.IsUnknown() {
		return input
	}

	// Convert input to map[string]interface{} via JSON marshaling/unmarshaling
	inputMap := utils.AttrValueToInterface(input.UnderlyingValue())
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

	// Use type-hinted conversion to preserve Set types from original input
	return types.DynamicValue(utils.InterfaceToAttrValueWithTypeHint(merged, input.UnderlyingValue()))
}

func (r *customCrudResource) mergeInputWithWO(input types.Dynamic, inputWO types.String) interface{} {
	var inputMap map[string]interface{}
	if !input.IsNull() && !input.IsUnknown() {
		if m, ok := utils.AttrValueToInterface(input.UnderlyingValue()).(map[string]interface{}); ok {
			inputMap = m
		}
	}
	if inputMap == nil {
		inputMap = make(map[string]interface{})
	}

	merged := make(map[string]interface{})
	for k, v := range inputMap {
		merged[k] = v
	}

	if !inputWO.IsNull() && !inputWO.IsUnknown() {
		var woMap map[string]interface{}
		if err := json.Unmarshal([]byte(inputWO.ValueString()), &woMap); err == nil {
			for k, v := range woMap {
				merged[k] = v
			}
		}
	}

	return merged
}
