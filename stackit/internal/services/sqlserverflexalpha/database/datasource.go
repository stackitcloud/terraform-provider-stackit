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
	sqlserverflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/database/datasources_gen"
	sqlserverflexUtils "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/utils"
)

var _ datasource.DataSource = (*databaseDataSource)(nil)

func NewDatabaseDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

type databaseDataSource struct {
	client       *sqlserverflexalpha.APIClient
	providerData core.ProviderData
}

func (d *databaseDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_database"
}

func (d *databaseDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = sqlserverflexalphaGen.DatabaseDataSourceSchema(ctx)
}

// Configure adds the provider configured client to the data source.
func (d *databaseDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
	tflog.Info(ctx, "SQL SERVER Flex database client configured")
}

func (d *databaseDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data sqlserverflexalphaGen.DatabaseModel

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
