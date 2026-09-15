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
	"github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"

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

// DataSourceModel is the internal model of the terraform data source
type DataSourceModel struct {
	Id                  types.String `tfsdk:"id"` // needed by TF
	ProjectId           types.String `tfsdk:"project_id"`
	RunnerId            types.String `tfsdk:"runner_id"`
	IntakeId            types.String `tfsdk:"intake_id"`
	DisplayName         types.String `tfsdk:"display_name"`
	Description         types.String `tfsdk:"description"`
	Labels              types.Map    `tfsdk:"labels"`
	Region              types.String `tfsdk:"region"`
	Uri                 types.String `tfsdk:"uri"`
	Topic               types.String `tfsdk:"topic"`
	DeadLetterTopic     types.String `tfsdk:"dead_letter_topic"`
	CreateTime          types.String `tfsdk:"create_time"`
	DremioTokenEndpoint types.String `tfsdk:"dremio_token_endpoint"`
	CatalogAuthType     types.String `tfsdk:"catalog_auth_type"`
	CatalogNamespace    types.String `tfsdk:"catalog_namespace"`
	CatalogPartitioning types.String `tfsdk:"catalog_partitioning"`
	CatalogPartitionBy  types.List   `tfsdk:"catalog_partition_by"`
	CatalogTableName    types.String `tfsdk:"catalog_table_name"`
	CatalogUri          types.String `tfsdk:"catalog_uri"`
	CatalogWarehouse    types.String `tfsdk:"catalog_warehouse"`
}

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
	resp.Schema = schema.Schema{
		Description: descriptions["datasource_main"],
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
			"display_name": schema.StringAttribute{
				Description: descriptions["display_name"],
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
			"topic": schema.StringAttribute{
				Description: descriptions["topic"],
				Computed:    true,
			},
			"dead_letter_topic": schema.StringAttribute{
				Description: descriptions["dead_letter_topic"],
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
	var model DataSourceModel
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

	err = mapDataSourceFields(ctx, intakeResp, &model, region)
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

func mapDataSourceFields(ctx context.Context, intakeResp *intake.IntakeResponse, model *DataSourceModel, region string) error {
	if intakeResp == nil {
		return fmt.Errorf("response input is nil")
	}
	if model == nil {
		return fmt.Errorf("model input is nil")
	}

	model.Id = utils.BuildInternalTerraformId(
		model.ProjectId.ValueString(),
		region,
		intakeResp.Id,
	)

	labels, err := utils.MapLabels(ctx, &intakeResp.Labels, model.Labels)
	if err != nil {
		return err
	}

	model.IntakeId = types.StringValue(intakeResp.Id)
	model.RunnerId = types.StringValue(intakeResp.IntakeRunnerId)
	model.DisplayName = types.StringValue(intakeResp.DisplayName)
	model.Labels = labels
	model.Description = types.StringPointerValue(intakeResp.Description)
	model.Region = types.StringValue(region)
	model.Uri = types.StringValue(intakeResp.Uri)
	model.Topic = types.StringValue(intakeResp.Topic)
	model.DeadLetterTopic = types.StringValue(intakeResp.DeadLetterTopic)
	model.CreateTime = types.StringValue(intakeResp.CreateTime.String())

	model.CatalogNamespace = types.StringPointerValue(intakeResp.Catalog.Namespace)
	model.CatalogTableName = types.StringPointerValue(intakeResp.Catalog.TableName)
	model.CatalogUri = types.StringValue(intakeResp.Catalog.Uri)
	model.CatalogWarehouse = types.StringValue(intakeResp.Catalog.Warehouse)

	if intakeResp.Catalog.Partitioning != nil {
		model.CatalogPartitioning = types.StringValue(string(*intakeResp.Catalog.Partitioning))
	} else {
		model.CatalogPartitioning = types.StringNull()
	}

	if intakeResp.Catalog.PartitionBy != nil {
		partitionByList, diags := types.ListValueFrom(ctx, types.StringType, intakeResp.Catalog.PartitionBy)
		if diags.HasError() {
			return fmt.Errorf("converting partition_by list: %v", diags)
		}
		model.CatalogPartitionBy = partitionByList
	} else {
		model.CatalogPartitionBy = types.ListNull(types.StringType)
	}

	if intakeResp.Catalog.Auth != nil {
		model.CatalogAuthType = types.StringValue(string(intakeResp.Catalog.Auth.Type))
		if intakeResp.Catalog.Auth.Dremio != nil {
			model.DremioTokenEndpoint = types.StringValue(intakeResp.Catalog.Auth.Dremio.TokenEndpoint)
		} else {
			model.DremioTokenEndpoint = types.StringNull()
		}
	} else {
		model.CatalogAuthType = types.StringNull()
		model.DremioTokenEndpoint = types.StringNull()
	}

	return nil
}
