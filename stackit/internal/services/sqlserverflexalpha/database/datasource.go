package sqlserverflexalpha

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	sqlserverflexalphaGen "github.com/mhenselin/terraform-provider-stackitprivatepreview/stackit/internal/services/sqlserverflexalpha/database/datasources_gen"
)

var _ datasource.DataSource = (*databaseDataSource)(nil)

func NewDatabaseDataSource() datasource.DataSource {
	return &databaseDataSource{}
}

type databaseDataSource struct{}

type databaseDataSourceModel struct {
	Id types.String `tfsdk:"id"`
}

func (d *databaseDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sqlserverflexalpha_database"
}

func (d *databaseDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = sqlserverflexalphaGen.DatabaseDataSourceSchema(ctx)
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
