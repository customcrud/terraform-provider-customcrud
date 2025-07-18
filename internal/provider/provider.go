// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package provider

import (
	"context"

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
	version     string
	parallelism int
	semaphore   chan struct{} // nil if no limit
}

// CustomCRUDProviderModel describes the provider data model.
type CustomCRUDProviderModel struct {
	// Provider-level configuration can be added here if needed
	Parallelism types.Int64 `tfsdk:"parallelism"`
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
			// Provider configuration attributes can be added here
		},
	}
}

func (p *CustomCRUDProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data CustomCRUDProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...) // get config into data

	if resp.Diagnostics.HasError() {
		return
	}

	if data.Parallelism.IsNull() || data.Parallelism.IsUnknown() {
		p.parallelism = 0
	} else {
		p.parallelism = int(data.Parallelism.ValueInt64())
	}
	if p.parallelism > 0 {
		p.semaphore = make(chan struct{}, p.parallelism)
	} else {
		p.semaphore = nil // unlimited
	}

	// Pass the semaphore to resources via ResourceData
	resp.ResourceData = p.semaphore
}

func (p *CustomCRUDProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewCustomCrudResource,
	}
}

func (p *CustomCRUDProvider) EphemeralResources(ctx context.Context) []func() ephemeral.EphemeralResource {
	return []func() ephemeral.EphemeralResource{}
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
