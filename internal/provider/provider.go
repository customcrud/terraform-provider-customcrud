// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

	"github.com/customcrud/terraform-provider-customcrud/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/ephemeral"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ provider.Provider = &CustomCRUDProvider{}
var _ provider.ProviderWithFunctions = &CustomCRUDProvider{}
var _ provider.ProviderWithEphemeralResources = &CustomCRUDProvider{}

// CustomCRUDProvider defines the provider implementation.
type CustomCRUDProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
	config  utils.CustomCRUDProviderConfig
}

type CustomCRUDProviderModel struct {
	Parallelism          types.Int64   `tfsdk:"parallelism"`
	HighPrecisionNumbers types.Bool    `tfsdk:"high_precision_numbers"`
	DefaultInputs        types.Dynamic `tfsdk:"default_inputs"`
}

func (p *CustomCRUDProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "customcrud"
	resp.Version = p.version
}

func (p *CustomCRUDProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A provider that allows custom CRUD operations via subprocess calls.",
		Attributes: map[string]schema.Attribute{
			"parallelism": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Maximum number of scripts to execute in parallel. 0 means unlimited (default).",
			},
			"high_precision_numbers": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Enable high precision for floating point numbers. This will cause the json parsing for outputs to use 512-bit floats instead of the default 64-bit.",
			},
			"default_inputs": schema.DynamicAttribute{
				Optional:            true,
				MarkdownDescription: "Default input values merged into every resource and data source input. Resource-level input takes priority over these defaults.",
			},
		},
	}
}

func (p *CustomCRUDProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CustomCRUDProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...) // get config into data
	if resp.Diagnostics.HasError() {
		return
	}

	p.config = utils.CustomCRUDProviderConfigDefaults()

	if !data.Parallelism.IsNull() && !data.Parallelism.IsUnknown() {
		p.config.Parallelism = int(data.Parallelism.ValueInt64())
	}

	if p.config.Parallelism > 0 {
		p.config.Semaphore = make(chan struct{}, p.config.Parallelism)
	}

	if !data.HighPrecisionNumbers.IsNull() {
		p.config.HighPrecisionNumbers = data.HighPrecisionNumbers.ValueBool()
	}

	if !data.DefaultInputs.IsNull() && !data.DefaultInputs.IsUnknown() {
		p.config.DefaultInputs = utils.AttrValueToInterface(data.DefaultInputs.UnderlyingValue())
	}

	resp.ResourceData = p
	resp.DataSourceData = p
}

func (p *CustomCRUDProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomCrudResource,
	}
}

func (p *CustomCRUDProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{
		NewCustomCrudEphemeral,
	}
}

func (p *CustomCRUDProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewCustomCrudDataSource,
	}
}

func (p *CustomCRUDProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &CustomCRUDProvider{
			version: version,
		}
	}
}
