// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: Apache-2.0

package sqlserverflexalpha

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/pkg/sqlserverflexalpha"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/conversion"
	"github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/core"
	sqlserverflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/utils"

	sqlserverflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/flavors/datasources_gen"
)

var _ datasource.DataSource = (*flavorsDataSource)(nil)

func NewFlavorsDataSource() datasource.DataSource {
	return &flavorsDataSource{}
}

type flavorsDataSource struct {
	client       *sqlserverflexalpha.APIClient
	providerData core.ProviderData
}

func (d *flavorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_flavors"
}

func (d *flavorsDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = sqlserverflexalphaGen.FlavorsDataSourceSchema(ctx)
}

// Configure adds the provider configured client to the data source.
func (d *flavorsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := sqlserverflexUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "SQL SERVER Flex flavors client configured")
}

func (d *flavorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sqlserverflexalphaGen.FlavorsModel

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
