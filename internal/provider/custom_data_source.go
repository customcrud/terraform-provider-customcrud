package provider

import (
	"context"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &customCrudDataSource{}
var _ datasource.DataSourceWithConfigure = &customCrudDataSource{}

type customCrudDataSourceModel struct {
	Hooks  types.List    `tfsdk:"hooks"`
	Input  types.Dynamic `tfsdk:"input"`
	Output types.Dynamic `tfsdk:"output"`
}

func (m *customCrudDataSourceModel) GetHooks() types.List {
	return m.Hooks
}

type customCrudDataSource struct {
	config utils.CustomCRUDProviderConfig
}

func NewCustomCrudDataSource() datasource.DataSource {
	return &customCrudDataSource{}
}

func (d *customCrudDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = "customcrud"
}

func (d *customCrudDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"input": schema.DynamicAttribute{
				Optional:    true,
				Description: "Input data for the data source",
			},
			"output": schema.DynamicAttribute{
				Computed:    true,
				Description: "Output data from the data source",
			},
		},
		Blocks: map[string]schema.Block{
			"hooks": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						utils.Read: schema.StringAttribute{
							Required:    true,
							Description: "Read command (space-separated command and arguments)",
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

func (d *customCrudDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		d.config = utils.CustomCRUDProviderConfigDefaults()
		return
	}
	if data, ok := req.ProviderData.(*CustomCRUDProvider); ok {
		d.config = data.config
	}
}

func (d *customCrudDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	utils.WithSemaphore(d.config.Semaphore, func() {
		var data customCrudDataSourceModel
		resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
		if resp.Diagnostics.HasError() {
			return
		}

		payload := utils.ExecutionPayload{
			Input: utils.MergeDefaultInputs(d.config, utils.AttrValueToInterface(data.Input.UnderlyingValue())),
		}
		result, ok := utils.RunCrudScript(ctx, d.config, &data, payload, &resp.Diagnostics, utils.CrudRead)
		if !ok {
			return
		}

		data.Output = utils.MapToDynamic(result.Result)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	})
}
