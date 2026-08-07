package intakes

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/stackitcloud/stackit-sdk-go/core/oapierror"

	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/conversion"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	intakeUtils "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/services/intake/utils"
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/validate"

	intake "github.com/stackitcloud/stackit-sdk-go/services/intake/v1betaapi"
)

// Ensure the implementation satisfies the expected interfaces
var (
	_ datasource.DataSource = &intakesDataSource{}
)

// NewIntakesDataSource is a helper function to simplify the provider implementation
func NewIntakesDataSource() datasource.DataSource {
	return &intakesDataSource{}
}

type intakesDataSource struct {
	client       *intake.APIClient
	providerData core.ProviderData
}

func (d *intakesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_intakes"
}

// Configure adds the provider configured client to the data source
func (d *intakesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	var ok bool
	d.providerData, ok = conversion.ParseProviderData(ctx, req.ProviderData, &resp.Diagnostics)
	if !ok {
		return
	}

	apiClient := intakeUtils.ConfigureClient(ctx, &d.providerData, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	d.client = apiClient
	tflog.Info(ctx, "Intakes client configured for data source")
}

// Schema defines the schema for the data source
func (d *intakesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	descriptions := map[string]string{ //nolint:gosec // descriptions
		"main":                         "Datasource for STACKIT Intake.",
		"id":                           "Terraform's internal resource identifier. It is structured as `project_id`,`region`,`intake_id`.",
		"project_id":                   "STACKIT Project ID to which the intake is associated.",
		"intake_id":                    "The intake ID.",
		"runner_id":                    "The runner ID.",
		"name":                         "The name of the intake.",
		"description":                  "The description of the intake.",
		"labels":                       "User-defined labels.",
		"uri":                          "The URI of the intake.",
		"create_time":                  "The creation time of the intake.",
		"region":                       "The resource region. If not defined, the provider region is used.",
		"dremio_personal_access_token": "The Dremio personal access token.",
		"dremio_token_endpoint":        "The Dremio token endpoint.",
		"catalog_auth_type":            "The catalog authentication type.",
		"catalog_namespace":            "The catalog namespace.",
		"catalog_partitioning":         "The catalog partitioning.",
		"catalog_partition_by":         "The catalog partition by.",
		"catalog_table_name":           "The catalog table name.",
		"catalog_uri":                  "The catalog URI.",
		"catalog_warehouse":            "The catalog warehouse.",
	}

	resp.Schema = schema.Schema{
		Description: descriptions["main"],
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: descriptions["id"],
				Computed:    true,
			},
			"project_id": schema.StringAttribute{
				Description: descriptions["project_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"intake_id": schema.StringAttribute{
				Description: descriptions["intake_id"],
				Required:    true,
				Validators: []validator.String{
					validate.UUID(),
					validate.NoSeparator(),
				},
			},
			"runner_id": schema.StringAttribute{
				Description: descriptions["runner_id"],
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: descriptions["name"],
				Computed:    true,
			},
			"description": schema.StringAttribute{
				Description: descriptions["description"],
				Computed:    true,
			},
			"labels": schema.MapAttribute{
				Description: descriptions["labels"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"uri": schema.StringAttribute{
				Description: descriptions["uri"],
				Computed:    true,
			},
			"create_time": schema.StringAttribute{
				Description: descriptions["create_time"],
				Computed:    true,
			},
			"region": schema.StringAttribute{
				Optional:    true,
				Description: descriptions["region"],
			},
			"dremio_personal_access_token": schema.StringAttribute{
				Description: descriptions["dremio_personal_access_token"],
				Computed:    true,
				Sensitive:   true,
			},
			"dremio_token_endpoint": schema.StringAttribute{
				Description: descriptions["dremio_token_endpoint"],
				Computed:    true,
			},
			"catalog_auth_type": schema.StringAttribute{
				Description: descriptions["catalog_auth_type"],
				Computed:    true,
			},
			"catalog_namespace": schema.StringAttribute{
				Description: descriptions["catalog_namespace"],
				Computed:    true,
			},
			"catalog_partitioning": schema.StringAttribute{
				Description: descriptions["catalog_partitioning"],
				Computed:    true,
			},
			"catalog_partition_by": schema.ListAttribute{
				Description: descriptions["catalog_partition_by"],
				ElementType: types.StringType,
				Computed:    true,
			},
			"catalog_table_name": schema.StringAttribute{
				Description: descriptions["catalog_table_name"],
				Computed:    true,
			},
			"catalog_uri": schema.StringAttribute{
				Description: descriptions["catalog_uri"],
				Computed:    true,
			},
			"catalog_warehouse": schema.StringAttribute{
				Description: descriptions["catalog_warehouse"],
				Computed:    true,
			},
		},
	}
}

// Read refreshes the Terraform state with the latest data.
func (d *intakesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) { // nolint:gocritic // function signature required by Terraform
	var model Model
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx = core.InitProviderContext(ctx)

	projectId := model.ProjectId.ValueString()
	region := d.providerData.GetRegionWithOverride(model.Region)
	intakeId := model.IntakeId.ValueString()
	ctx = tflog.SetField(ctx, "project_id", projectId)
	ctx = tflog.SetField(ctx, "region", region)
	ctx = tflog.SetField(ctx, "intake_id", intakeId)

	intakeResp, err := d.client.DefaultAPI.GetIntake(ctx, projectId, region, intakeId).Execute()
	if err != nil {
		var oapiErr *oapierror.GenericOpenAPIError
		if errors.As(err, &oapiErr) {
			if oapiErr.StatusCode == http.StatusNotFound {
				core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading intake", fmt.Sprintf("Intake with ID %s not found in project %s and region %s", intakeId, projectId, region))
				return
			}
		}
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading intake", fmt.Sprintf("Calling API: %v", err))
		return
	}

	ctx = core.LogResponse(ctx)

	err = mapFields(ctx, intakeResp, &model, region)
	if err != nil {
		core.LogAndAddError(ctx, &resp.Diagnostics, "Error reading intake", fmt.Sprintf("Processing API payload: %v", err))
		return
	}

	// Set refreshed state
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Info(ctx, "Intake read")
}
