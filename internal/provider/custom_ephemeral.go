package provider

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ ephemeral.EphemeralResource = &customCrudEphemeral{}
var _ ephemeral.EphemeralResourceWithConfigure = &customCrudEphemeral{}
var _ ephemeral.EphemeralResourceWithRenew = &customCrudEphemeral{}
var _ ephemeral.EphemeralResourceWithClose = &customCrudEphemeral{}

type privater interface {
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}

type customCrudEphemeralModel struct {
	Hooks  types.List    `tfsdk:"hooks"`
	Input  types.Dynamic `tfsdk:"input"`
	Output types.Dynamic `tfsdk:"output"`
}

func (m *customCrudEphemeralModel) GetHooks() types.List {
	return m.Hooks
}

type customCrudEphemeral struct {
	config utils.CustomCRUDProviderConfig
}

func NewCustomCrudEphemeral() ephemeral.EphemeralResource {
	return &customCrudEphemeral{}
}

func (e *customCrudEphemeral) Metadata(ctx context.Context, req ephemeral.MetadataRequest, resp *ephemeral.MetadataResponse) {
	resp.TypeName = "customcrud"
}

func (e *customCrudEphemeral) Schema(ctx context.Context, req ephemeral.SchemaRequest, resp *ephemeral.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"input": schema.DynamicAttribute{
				Optional:    true,
				Description: "Input data for the ephemeral resource",
			},
			"output": schema.DynamicAttribute{
				Computed:    true,
				Description: "Output data from the ephemeral resource",
			},
		},
		Blocks: map[string]schema.Block{
			"hooks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						utils.Open: schema.StringAttribute{
							Required:    true,
							Description: "Open command (space-separated command and arguments)",
						},
						utils.Renew: schema.StringAttribute{
							Optional:    true,
							Description: "Renew command (space-separated command and arguments)",
						},
						utils.Close: schema.StringAttribute{
							Optional:    true,
							Description: "Close command (space-separated command and arguments)",
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

func (e *customCrudEphemeral) Configure(ctx context.Context, req ephemeral.ConfigureRequest, resp *ephemeral.ConfigureResponse) {
	if req.ProviderData == nil {
		e.config = utils.CustomCRUDProviderConfigDefaults()
		return
	}
	if data, ok := req.ProviderData.(*CustomCRUDProvider); ok {
		e.config = data.config
	}
}

func (e *customCrudEphemeral) Open(ctx context.Context, req ephemeral.OpenRequest, resp *ephemeral.OpenResponse) {
	utils.WithSemaphore(e.config.Semaphore, func() {
		var data customCrudEphemeralModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		payload := utils.ExecutionPayload{
			Input: utils.AttrValueToInterface(data.Input.UnderlyingValue()),
		}
		result, ok := utils.RunCrudScript(ctx, e.config, &data, payload, &resp.Diagnostics, utils.CrudOpen)
		if !ok {
			return
		}

		data.Output = utils.MapToDynamic(result.Result)
		resp.Diagnostics.Append(resp.Result.Set(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		// Save to private state for Renew/Close
		// Use plain Go types for JSON marshaling instead of framework types
		// data.Hooks is a list (nested block), we take the first element if it exists
		var hooksData interface{}
		if hooksList, ok := utils.AttrValueToInterface(data.Hooks).([]interface{}); ok && len(hooksList) > 0 {
			hooksData = hooksList[0]
		}
		hooksBytes, err := json.Marshal(hooksData)
		if err == nil && len(hooksBytes) > 0 {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, "hooks", hooksBytes)...)
		}

		inputBytes, err := json.Marshal(utils.AttrValueToInterface(data.Input.UnderlyingValue()))
		if err == nil && len(inputBytes) > 0 {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, "input", inputBytes)...)
		}

		outputBytes, err := json.Marshal(utils.AttrValueToInterface(data.Output.UnderlyingValue()))
		if err == nil && len(outputBytes) > 0 {
			resp.Diagnostics.Append(resp.Private.SetKey(ctx, "output", outputBytes)...)
		}
	})
}

func (e *customCrudEphemeral) Renew(ctx context.Context, req ephemeral.RenewRequest, resp *ephemeral.RenewResponse) {
	e.renew(ctx, req.Private, &resp.Diagnostics)
}

func (e *customCrudEphemeral) renew(ctx context.Context, priv privater, diagnostics *diag.Diagnostics) {
	utils.WithSemaphore(e.config.Semaphore, func() {
		// Get hooks from private state
		hooksBytes, diags := priv.GetKey(ctx, "hooks")
		diagnostics.Append(diags...)
		if diagnostics.HasError() || len(hooksBytes) == 0 {
			return
		}

		// Use a plain struct for unmarshaling to avoid types.String unmarshal issues
		var hooks struct {
			Renew string `json:"renew"`
		}
		if err := json.Unmarshal(hooksBytes, &hooks); err != nil {
			diagnostics.AddError("Failed to unmarshal hooks from private state", err.Error())
			return
		}

		// If no renew hook, just return (nothing to do)
		if hooks.Renew == "" {
			return
		}

		// Get input/output from private state
		inputBytes, diags := priv.GetKey(ctx, "input")
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}

		outputBytes, diags := priv.GetKey(ctx, "output")
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}

		var input, output interface{}
		if len(inputBytes) > 0 {
			_ = json.Unmarshal(inputBytes, &input)
		}
		if len(outputBytes) > 0 {
			_ = json.Unmarshal(outputBytes, &output)
		}

		payload := utils.ExecutionPayload{
			Input:  input,
			Output: output,
		}

		cmd := strings.Fields(hooks.Renew)
		if len(cmd) == 0 {
			return
		}

		_, err := utils.Execute(ctx, e.config, cmd, payload)
		if err != nil {
			diagnostics.AddError("Renew Script Failed", err.Error())
		}
		// Renew cannot update result data per Terraform spec, just extends renewal
	})
}

func (e *customCrudEphemeral) Close(ctx context.Context, req ephemeral.CloseRequest, resp *ephemeral.CloseResponse) {
	e.close(ctx, req.Private, &resp.Diagnostics)
}

func (e *customCrudEphemeral) close(ctx context.Context, priv privater, diagnostics *diag.Diagnostics) {
	utils.WithSemaphore(e.config.Semaphore, func() {
		// Get hooks from private state
		hooksBytes, diags := priv.GetKey(ctx, "hooks")
		diagnostics.Append(diags...)
		if diagnostics.HasError() || len(hooksBytes) == 0 {
			return
		}

		// Use a plain struct for unmarshaling
		var hooks struct {
			Close string `json:"close"`
		}
		if err := json.Unmarshal(hooksBytes, &hooks); err != nil {
			diagnostics.AddError("Failed to unmarshal hooks from private state", err.Error())
			return
		}

		// If no close hook, just return (nothing to do)
		if hooks.Close == "" {
			return
		}

		// Get input/output from private state
		inputBytes, diags := priv.GetKey(ctx, "input")
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}

		outputBytes, diags := priv.GetKey(ctx, "output")
		diagnostics.Append(diags...)
		if diagnostics.HasError() {
			return
		}

		var input, output interface{}
		if len(inputBytes) > 0 {
			_ = json.Unmarshal(inputBytes, &input)
		}
		if len(outputBytes) > 0 {
			_ = json.Unmarshal(outputBytes, &output)
		}

		payload := utils.ExecutionPayload{
			Input:  input,
			Output: output,
		}

		cmd := strings.Fields(hooks.Close)
		if len(cmd) == 0 {
			return
		}

		_, _ = utils.Execute(ctx, e.config, cmd, payload)
	})
}
