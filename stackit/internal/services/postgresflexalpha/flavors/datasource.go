// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package postgresflexalpha

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/postgresflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	postgresflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/flavors/datasources_gen"
	postgresflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/postgresflexalpha/utils"
)

var (
	_ datasource.DataSource              = &flavorsDataSource{}
	_ datasource.DataSourceWithConfigure = &flavorsDataSource{}
)

func NewFlavorsDataSource() datasource.DataSource {
	return &flavorsDataSource{}
}

type flavorsDataSource struct {
	client       *postgresflexalpha.APIClient
	providerData core.ProviderData
}

func (d *flavorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresflexalpha_flavors"
}

func (d *flavorsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = postgresflexalphaGen.FlavorsDataSourceSchema(ctx)
}

// Configure adds the provider configured client to the data source.
func (d *flavorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := postgresflexUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Postgres Flex version client configured")
}

func (d *flavorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data postgresflexalphaGen.FlavorsModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Todo: Read API call logic

	// Example data value setting
	// data.Id = types.StringValue("example-id")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
